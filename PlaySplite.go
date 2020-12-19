package verifyjc

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
)

//实现对象池应对彩票效率
var playSplitePool = sync.Pool{
	New: func() interface{} {
		return &PlaySplite{}
	},
}

type PlaySplite struct {
	Match    []Match  //赛程数组
	Playtype []string //串关方式
	Multiple int      //倍数
	Dan      []string //设胆
	Lotid    int      //彩种ID
}

func (play *PlaySplite) SetSpliteTicket(match []Match, playtype []string, multiple int, dan []string, lotid int) {
	play.Match = match
	play.Playtype = playtype
	play.Multiple = multiple
	play.Dan = dan
	play.Lotid = lotid
}

//分析并组合数据结构
func (play *PlaySplite) GetZuJcTicket() ([]JcTicket, int, int, error) {
	var allTicket []JcTicket
	var jcTicket []JcTicket
	var jcGroupTicket []JcTicket
	matchChuan, matchChuanGroup, analysisError := play.analysisPlay()
	if analysisError != nil {
		return allTicket, 0, 0, analysisError
	}
	if len(play.Match) < 9 && len(play.Dan) == 0 && len(matchChuan) > 0 && play.Lotid != JcConf.Htcode && play.Lotid != JcConf.LqHtCode {
		jcTicket = play.FreeManyMatchTicket(matchChuan) //自由过关

	} else if len(matchChuan) > 0 {
		jcTicket = play.freeZuJcTicket(matchChuan) //自由过关
	}

	if len(matchChuanGroup) > 0 {
		jcGroupTicket = play.groupZuJcTicket(matchChuanGroup) //组合过关
	}

	if len(jcTicket) > 0 {
		allTicket = append(allTicket, jcTicket...)
	}

	if len(jcGroupTicket) > 0 {
		allTicket = append(allTicket, jcGroupTicket...)
	}

	return play.matchDanSplite(allTicket) //处理设胆和拆票
}

//分析串关方式
func (play *PlaySplite) analysisPlay() ([]string, []string, error) {
	var matchChuan []string = make([]string, 0)
	var matchChuanGroup []string = make([]string, 0)
	var matchnum int = len(play.Match)

	//判断串关方式- 组合过关和自由过关
	var playtypenum = len(play.Playtype)
	var matchChuanNum []int = make([]int, playtypenum)
	var checkMums int = 0
	for i := 0; i < playtypenum; i++ {

		//自由过关
		if v, ok := JcConf.MatchChuan[play.Playtype[i]]; ok && v <= matchnum {
			matchChuan = append(matchChuan, play.Playtype[i])
			matchChuanNum[i] = v
			checkMums++
		}

		//组合过关
		if v, ok := JcConf.MatchChuanGroup[play.Playtype[i]]; ok && v <= matchnum {
			matchChuanGroup = append(matchChuanGroup, play.Playtype[i])
			matchChuanNum[i] = v
			checkMums++
		}

	}

	return matchChuan, matchChuanGroup, nil
}

//自由过关 - 拆票，计算票数，注数
func (play *PlaySplite) freeZuJcTicket(matchChuan []string) []JcTicket {
	var match []interface{} = play.matchToInterface()
	var ticketAll []JcTicket = make([]JcTicket, 0)

	for _, v := range matchChuan {
		ticketAllOne := play.freeCalculationZhu(match, v)
		ticketAll = append(ticketAll, ticketAllOne...)
	}
	return ticketAll
}

//组合过关-拆票，计算票数，注数
func (play *PlaySplite) groupZuJcTicket(matchGroupChuan []string) []JcTicket {
	var match []interface{} = play.matchToInterface()
	var ticketAll []JcTicket = make([]JcTicket, 0)
	for _, v := range matchGroupChuan {
		var nums int = JcConf.MatchChuanGroup[v]
		zuhe := NewZuhe(match, nums)
		matchres := zuhe.FindNumsByIndexs()

		//开始组合票
		for _, vt := range matchres { //每次循环就是一张票
			var tempCode []string
			var tempLotNum int = 0
			var tempIssue string
			var vtNum int = len(vt)
			tempCode = make([]string, vtNum)
			var tempMatchCode []string = make([]string, vtNum)
			var tempLotidArr []string = make([]string, vtNum)
			var tempNotHunTou []string = make([]string, vtNum)
			var tempSpCode []string = make([]string, vtNum)
			var tempSpMatchCode []string = make([]string, vtNum)
			for km, vm := range vt {
				matchObj := vm.(Match)
				tempCode[km] = matchObj.Betcode
				tempLotidArr[km] = matchObj.Lotid
				tempMatchCode[km] = matchObj.Matchcode
				tempNotHunTou[km] = matchObj.Issue
				tempSpCode[km] = fmt.Sprintf("%s:%s(%s)", matchObj.Lotid, matchObj.Issue, matchObj.MatchSp)
				tempSpMatchCode[km] = fmt.Sprintf("%s(%s)", matchObj.Issue, matchObj.MatchSp)

				if km == (vtNum - 1) {
					tempIssue = fmt.Sprintf("%s%s", "20", strings.Replace(matchObj.Issue, "-", "", -1))
				}
			}

			//一张票里面不能含有相同场次的票  需要把相同场次的票给去除掉
			sort.Strings(tempNotHunTou)
			var checkStatus bool = false
			for i := 0; i < len(tempNotHunTou); i++ {
				if i > 0 {
					if tempNotHunTou[i] == tempNotHunTou[i-1] {
						checkStatus = true
						break
					}
				}
			}

			if checkStatus {
				continue
			}

			//重新计算注数
			var newPlayType []string = JcConf.ManyChuan[v]

			for _, vplaytype := range newPlayType {
				var ticketZhuOne int = play.groupCalculationZhu(vt, vplaytype)
				tempLotNum = tempLotNum + ticketZhuOne
			}

			//修正部分混投的票
			var lotcodeTemp string = strings.Join(tempCode, ";")
			var lotSpTemp string = strings.Join(tempSpMatchCode, ";")
			var lotidTemp int = play.Lotid
			if play.Lotid == JcConf.Htcode || play.Lotid == JcConf.LqHtCode {
				lotSpTemp = strings.Join(tempSpCode, ";")
				var lotUniqueids []string = play.arrayUnique(tempLotidArr)
				if len(lotUniqueids) == 1 {
					lotidTemp, _ = strconv.Atoi(lotUniqueids[0])
					lotcodeTemp = strings.Join(tempMatchCode, ";")
					lotSpTemp = strings.Join(tempSpMatchCode, ";")
				}
			}

			jcticket := JcTicket{
				Lotnum:   lotcodeTemp,
				Issue:    tempIssue,
				Money:    tempLotNum * 2 * play.Multiple,
				Multiple: play.Multiple,
				BetNum:   tempLotNum,
				Lotid:    lotidTemp,
				Playtype: v,
				Lotres:   lotSpTemp,
			}
			ticketAll = append(ticketAll, jcticket)
		}
	}
	return ticketAll
}

//过关计算注数 -- 组合过关
func (play *PlaySplite) groupCalculationZhu(match []interface{}, playtype string) int {
	var ticketZhu int = 0
	var num int = JcConf.MatchChuan[playtype]
	zuhe := NewZuhe(match, num)
	matchres := zuhe.FindNumsByIndexs()
	for _, vt := range matchres {
		var tempLotNum int = 1
		for _, vm := range vt {
			matchObj := vm.(Match)
			tempLotNum = tempLotNum * matchObj.Betlen
		}
		tempLotNum = tempLotNum
		ticketZhu = ticketZhu + tempLotNum
	}

	return ticketZhu
}

//过关计算注数 -- 自由过关
func (play *PlaySplite) freeCalculationZhu(match []interface{}, playtype string) []JcTicket {
	var ticketAll []JcTicket = make([]JcTicket, 0)
	var num int = JcConf.MatchChuan[playtype]

	zuhe := NewZuhe(match, num)
	matchres := zuhe.FindNumsByIndexs()

	for _, vt := range matchres {
		var tempLotNum int = 1
		var tempIssue string
		var matchNum int = len(vt)
		var checkStatus bool = true
		var tempCode []string = make([]string, matchNum)

		var tempMatchCode []string = make([]string, matchNum)
		var tempSpCode []string = make([]string, matchNum)
		var tempSpMatchCode []string = make([]string, matchNum)

		//混投有些需要特殊处理成普通玩法
		var tempLotidArr []string = make([]string, matchNum)
		for i := 0; i < matchNum; i++ {
			matchObj := vt[i].(Match)
			if i > 0 {
				matchObj1 := vt[i-1].(Match)
				if strings.Compare(matchObj.Issue, matchObj1.Issue) == 0 {
					checkStatus = false
					continue
				}

			}
			tempLotidArr[i] = matchObj.Lotid
			tempCode[i] = matchObj.Betcode
			tempMatchCode[i] = matchObj.Matchcode
			tempSpCode[i] = fmt.Sprintf("%s:%s(%s)", matchObj.Lotid, matchObj.Issue, matchObj.MatchSp)
			tempSpMatchCode[i] = fmt.Sprintf("%s(%s)", matchObj.Issue, matchObj.MatchSp)
			tempLotNum = tempLotNum * matchObj.Betlen
			if i == (matchNum - 1) {
				tempIssue = fmt.Sprintf("%s%s", "20", strings.Replace(matchObj.Issue, "-", "", -1))
			}
		}
		if !checkStatus {
			continue
		}

		//修正部分混投的票
		var lotcodeTemp string = strings.Join(tempCode, ";")
		var lotSpTemp string = strings.Join(tempSpMatchCode, ";")
		var lotidTemp int = play.Lotid
		if play.Lotid == JcConf.Htcode || play.Lotid == JcConf.LqHtCode {
			lotSpTemp = strings.Join(tempSpCode, ";")
			var lotUniqueids []string = play.arrayUnique(tempLotidArr)
			if len(lotUniqueids) == 1 {
				lotidTemp, _ = strconv.Atoi(lotUniqueids[0])
				lotcodeTemp = strings.Join(tempMatchCode, ";")
				lotSpTemp = strings.Join(tempSpMatchCode, ";")
			}
		}
		jcticket := JcTicket{
			Lotnum:   lotcodeTemp,
			Issue:    tempIssue,
			Money:    tempLotNum * 2 * play.Multiple,
			Multiple: play.Multiple,
			BetNum:   tempLotNum,
			Lotid:    lotidTemp,
			Playtype: playtype,
			Lotres:   lotSpTemp,
		}

		ticketAll = append(ticketAll, jcticket)
	}
	return ticketAll
}

/***********************************
*处理设胆 需要判断每张票是否在都含有的赛程
*例如 140904-003:140904-003:140904-003
***********************************/

func (play *PlaySplite) matchDan(ticketAll []JcTicket, ticketZhu int, ticketMoney int) ([]JcTicket, int, int) {
	if len(play.Dan) == 0 {
		return ticketAll, ticketZhu, ticketMoney
	}
	//票的数量
	var ticketAllNew []JcTicket = make([]JcTicket, 0)
	var ticketZhuNew int = 0
	var ticketMoneyNew int = 0
	var ticketNum int = len(ticketAll)
	for i := 0; i < ticketNum; i++ {
		var ticketCode string = ticketAll[i].Lotnum
		var ticketBool bool = true
		for _, v := range play.Dan {

			if strings.LastIndex(ticketCode, v) == -1 {
				ticketBool = false
				break
			}
		}
		if ticketBool {
			ticketAllNew = append(ticketAllNew, ticketAll[i])
			ticketZhuNew = ticketZhuNew + ticketAll[i].BetNum
			ticketMoneyNew = ticketMoneyNew + ticketAll[i].Money
		}

	}
	return ticketAllNew, ticketZhuNew, ticketMoneyNew
}

/********************************
*统一处理投注号码设胆和拆票
********************************/
func (play *PlaySplite) matchDanSplite(ticket []JcTicket) ([]JcTicket, int, int, error) {
	var ticketAll []JcTicket = make([]JcTicket, 0)
	var ticketOne []JcTicket
	var ticketZhu int = 0
	var ticketMoney int = 0
	var ticketAllNum int = len(ticket)
	var err error = nil
	for i := 0; i < ticketAllNum; i++ {
		var danRes bool = play.matchDanOne(ticket[i])
		if !danRes {
			continue
		}
		ticketOne, err = play.spliteByMoneyOne(ticket[i])

		if err != nil {
			continue
		}
		ticketZhu += ticket[i].BetNum
		ticketMoney += ticket[i].Money
		ticketAll = append(ticketAll, ticketOne...)
	}
	return ticketAll, ticketZhu, ticketMoney, err
}

/***********************************
*按照用户设置的比赛胆 --
*胆则每张票中必须含有
***********************************/
func (play *PlaySplite) matchDanOne(ticket JcTicket) bool {
	if len(play.Dan) == 0 {
		return true
	}
	var ticketCode string = ticket.Lotnum
	var ticketBool bool = true
	for _, v := range play.Dan {
		if strings.LastIndex(ticketCode, v) == -1 {
			ticketBool = false
			break
		}
	}
	return ticketBool
}

/***********************************
*按照注数和钱数拆票 -- 单张拆票
*单张票不能找过2万 不能超过99倍
***********************************/
func (play *PlaySplite) spliteByMoneyOne(ticket JcTicket) ([]JcTicket, error) {
	var jcTicket []JcTicket = make([]JcTicket, 0)
	var oneTicketZhu int = ticket.Money / ticket.Multiple / 2
	var oneTimes int = ticket.Multiple
	var newZhu int = 0
	if oneTicketZhu > JcConf.MaxZhu {
		return jcTicket, errors.New("一张票不能超过10000注")
	}

	if ticket.BetNum < JcConf.MaxZhu && (oneTicketZhu*JcConf.MaxMultiple < JcConf.MaxZhu && oneTimes <= JcConf.MaxMultiple) {
		jcTicket = append(jcTicket, ticket)
		return jcTicket, nil
	}

	//按照倍数拆票
	if oneTicketZhu*JcConf.MaxMultiple < JcConf.MaxZhu && ticket.Multiple > JcConf.MaxMultiple {
		newZhu = int(math.Ceil(float64(oneTimes) / float64(JcConf.MaxMultiple)))
		var newZhuSurplus int = oneTimes - (newZhu-1)*JcConf.MaxMultiple
		for i := 1; i < newZhu; i++ {
			ticket.Multiple = JcConf.MaxMultiple
			ticket.BetNum = oneTicketZhu
			ticket.Money = JcConf.MaxMultiple * oneTicketZhu * 2
			jcTicket = append(jcTicket, ticket)
		}
		//计算剩下的倍数
		ticket.Multiple = newZhuSurplus
		ticket.BetNum = oneTicketZhu
		ticket.Money = newZhuSurplus * oneTicketZhu * 2
		jcTicket = append(jcTicket, ticket)
		return jcTicket, nil
	}
	//按照钱数拆票
	var newBei int = JcConf.MaxZhu / oneTicketZhu //每张票多少倍
	newZhu = int(math.Ceil(float64(oneTimes) / float64(newBei)))
	var newZhuSurplus int = oneTimes - (newZhu-1)*newBei
	for i := 1; i < newZhu; i++ {
		ticket.Multiple = newBei
		ticket.BetNum = oneTicketZhu
		ticket.Money = newBei * oneTicketZhu * 2
		jcTicket = append(jcTicket, ticket)
	}
	//计算剩下的倍数
	ticket.Multiple = newZhuSurplus
	ticket.BetNum = oneTicketZhu
	ticket.Money = newZhuSurplus * oneTicketZhu * 2
	jcTicket = append(jcTicket, ticket)
	return jcTicket, nil
}

/***********************************
*match 转 interface
*转成interface数组传到组合类里面
***********************************/
func (play *PlaySplite) matchToInterface() []interface{} {
	var matchInter []interface{} = make([]interface{}, len(play.Match))
	for k, v := range play.Match {
		matchInter[k] = v
	}
	return matchInter
}

/*******************************************************
*用户没有设胆 并且用户投注的场次小于等于八场 -- 拆票
*则走自由过关的模式 一个串关方式多长比赛
*******************************************************/
func (play *PlaySplite) FreeManyMatchTicket(matchChuan []string) []JcTicket {
	var match []interface{} = play.matchToInterface()
	var jcTickets []JcTicket = make([]JcTicket, 0)
	var bets []string = make([]string, 0)
	var betSp []string = make([]string, 0)
	var issue []string = make([]string, 0)
	for _, vm := range play.Match {
		issue = append(issue, fmt.Sprintf("%s%s", "20", strings.Replace(vm.Issue, "-", "", -1)))
		bets = append(bets, vm.Matchcode)
		spd := fmt.Sprintf("%s(%s)", vm.Issue, vm.MatchSp)
		betSp = append(betSp, spd)

	}

	sort.Strings(issue)

	for _, playtype := range matchChuan {
		var zhushu int = play.groupCalculationZhu(match, playtype)
		jcticket := JcTicket{
			Lotnum:   strings.Join(bets, ";"),
			Issue:    issue[len(issue)-1],
			Money:    zhushu * 2 * play.Multiple,
			Multiple: play.Multiple,
			BetNum:   zhushu,
			Lotid:    play.Lotid,
			Playtype: playtype,
			Lotres:   strings.Join(betSp, ";"),
		}

		jcTickets = append(jcTickets, jcticket)
	}

	return jcTickets
}

func (play *PlaySplite) arrayUnique(a []string) (ret []string) {
	sort.Strings(a)
	a_len := len(a)
	for i := 0; i < a_len; i++ {
		if (i > 0 && a[i-1] == a[i]) || len(a[i]) == 0 {
			continue
		}
		ret = append(ret, a[i])
	}
	return
}

/**************************************
*@实例化竞彩组合类
*@match     赛程数组
*@playtype  串关方式
*@times     倍数
*@dan       设胆
*@lotid     彩种ID
***************************************/
func NewPlaySplite(match []Match, playtype []string, play int, multiple int, dan []string, lotid int) *PlaySplite {
	return &PlaySplite{
		Match:    match,
		Playtype: playtype,
		Multiple: multiple,
		Dan:      dan,
		Lotid:    lotid,
	}
}

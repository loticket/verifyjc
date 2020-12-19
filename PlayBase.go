package verifyjc

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type PlayBase struct {
	Lotnum   string //投注号码
	Money    int    //钱数
	Multiple int    //倍数
	BetNum   int    //注数
	Lotid    int    //彩种编号
	Playtype string //串关方式
	Dan      string //设置的胆
}

/**
 * @name:正则匹配投注信息 - 并验证投注信息
 * @msg:正则匹配的投注信息 - 并验证投注信息
 * @param betcode 投注字符串
 * @return: []Match 投注信息解析后的详细信息 []string 赛程ID  []string 投注玩法ID  error 验证投注信息是否错误
 */
func (play *PlayBase) MatchRegValidate() ([]Match, []string, []string, error) {
	var matchArr []string = strings.Split(play.Lotnum, ";")
	var match []Match = make([]Match, 0, len(matchArr))
	var matchId []string = make([]string, 0, len(matchArr))
	var lotIdArr []string = make([]string, 0)
	var tempMath Match
	var err error
	for k, v := range matchArr {
		err = nil
		if play.Lotid == JcConf.Htcode || play.Lotid == JcConf.LqHtCode {
			tempMath, err = play.matchRegHtOne(v)
		} else {
			tempMath, err = play.matchRegOne(v)
		}

		if err == nil {
			match = append(match, tempMath)
			var tempMathId string = fmt.Sprintf("20%s", strings.Replace(tempMath.Issue, "-", "", -1))
			if k == 0 {
				matchId = append(matchId, tempMathId)
				lotIdArr = append(lotIdArr, tempMath.Lotid)
			} else {
				matchIdExist := searchString(matchId, tempMathId)
				if !matchIdExist {
					matchId = append(matchId, tempMathId)
				}

				lotIdExist := searchString(lotIdArr, tempMath.Lotid)
				if !lotIdExist {
					lotIdArr = append(lotIdArr, tempMath.Lotid)
				}

			}

		} else {
			break
		}
	}

	if len(match) != len(matchArr) {
		return match, matchId, lotIdArr, errors.New("投注代码错误")
	}
	return match, matchId, lotIdArr, nil
}

/**
 * @name:验证串关方式
 * @msg:验证串关方式
 * @param lotIdArr 投注玩法id
 * @return: []string 过关数组 int 是不是单关 error 验证投注信息是否错误
 */
func (play *PlayBase) PlaytypeValidate(lotIdArr []string) ([]string, int, error) {
	var ptArr []string = strings.Split(play.Playtype, "^")
	var newPlaytype []string = play.arrayUnique(ptArr)
	var nums int = 0
	var single int = 0
	var maxChuan int = 0

	if len(newPlaytype) != len(ptArr) {
		return ptArr, single, errors.New("过关方式错误")
	}

	for _, v := range ptArr {
		if v == "1_1" {
			single = 1
		}

		//自由过关
		if zyNum, ok := JcConf.MatchChuan[v]; ok {
			nums++
			if zyNum > maxChuan {
				maxChuan = zyNum
			}
		}

		//组合过关
		if zhNum, ok := JcConf.MatchChuanGroup[v]; ok {
			nums++
			if zhNum > maxChuan {
				maxChuan = zhNum
			}
		}

	}

	//判断每种玩法的限制串关方式的最小值
	var lotidMaxMatch int = 0
	for k, lotid := range lotIdArr {
		maxMatch, _ := JcConf.MaxChuan[lotid]
		if k == 0 || (k != 0 && lotidMaxMatch < maxMatch) {
			lotidMaxMatch = maxMatch
		}

	}

	if lotidMaxMatch < maxChuan {
		return ptArr, single, errors.New("串关方式和赛程场次数比匹配")
	}

	if nums != len(ptArr) {
		return ptArr, single, errors.New("串关方式错误")
	}

	return ptArr, single, nil
}

/**
 * @name:正则匹配一场比赛的投注信息 - 并验证投注信息
 * @msg:正则匹配一场比赛的投注信息 - 并验证投注信息
 * @param betcode 投注字符串
 * @return: Match 投注信息解析后的详细信息 error 验证投注信息是否错误
 */
func (play *PlayBase) matchRegOne(betcode string) (Match, error) {
	reg, err := regexp.Compile("^(\\d{6})-(\\d{3})\\((.*)\\)$")
	if err != nil {
		return Match{}, errors.New("投注格式错误")
	}
	regRes := reg.FindStringSubmatch(betcode)
	if len(regRes) != 4 {
		return Match{}, errors.New("投注格式错误")
	}
	if strings.Compare(regRes[3], "") == 0 {
		return Match{}, errors.New("投注格式错误")
	}

	betres := strings.Split(regRes[3], ",")

	//判断投注格式是否正确
	var allBetCode []string = play.getBetCode(play.Lotid)
	if len(allBetCode) == 0 {
		return Match{}, errors.New("投注格式错误")
	}

	var valitate bool = true
	for key, betone := range betres {
		valitate = searchString(allBetCode, betone)
		if !valitate {
			break
		}
		if key > 0 && betres[key] == betres[key-1] {
			valitate = false
			break
		}
	}

	if !valitate {
		return Match{}, errors.New("投注格式错误")
	}

	return Match{
		Betcode:   betcode,
		Matchcode: betcode,
		Lotid:     strconv.Itoa(play.Lotid),
		Issue:     fmt.Sprintf("%s-%s", regRes[1], regRes[2]),
		Matchnums: regRes[2],
		Betnum:    regRes[3],
		Betnumarr: betres,
		Betlen:    len(betres),
	}, nil
}

/**
 * @name:正则匹配投注信息 - 并验证投注信息
 * @msg:正则匹配投注信息-只验证混合过关玩法 - 并验证投注信息
 * @param betcode 投注字符串
 * @return: Match 投注信息解析后的详细信息 error 验证投注信息是否错误
 */
func (play *PlayBase) matchRegHtOne(betcode string) (Match, error) {
	reg, err := regexp.Compile("^(\\d{2}):(\\d{6})-(\\d{3})\\((.*)\\)$")
	if err != nil {
		return Match{}, errors.New("投注格式错误")
	}
	regRes := reg.FindStringSubmatch(betcode)
	if len(regRes) != 5 {
		return Match{}, errors.New("投注格式错误")
	}
	if strings.Compare(regRes[4], "") == 0 {
		return Match{}, errors.New("投注格式错误")
	}

	betres := strings.Split(regRes[4], ",")
	lotid, _ := strconv.ParseInt(regRes[1], 10, 64)

	//判断投注格式是否正确
	var allBetCode []string = play.getBetCode(int(lotid))
	if len(allBetCode) == 0 {
		return Match{}, errors.New("投注格式错误")
	}
	var valitate bool = false
	for key, betone := range betres {
		valitate = searchString(allBetCode, betone)
		if !valitate {
			break
		}
		if key > 0 && betres[key] == betres[key-1] {
			valitate = false
			break
		}
	}

	if !valitate {
		return Match{}, errors.New("投注格式错误")
	}
	return Match{
		Betcode:   betcode,
		Matchcode: fmt.Sprintf("%s-%s(%s)", regRes[2], regRes[3], regRes[4]),
		Lotid:     regRes[1],
		Issue:     fmt.Sprintf("%s-%s", regRes[2], regRes[3]),
		Matchnums: regRes[3],
		Betnum:    regRes[4],
		Betnumarr: betres,
		Betlen:    len(betres),
	}, nil
}

/**
 * @name:检查设胆的正确性
 * @param matchIdArr []string  投注赛程的id
 * @return: error 错误信息
 */
func (play *PlayBase) CheckDan(matchIdArr []string) (error, []string) {

	if strings.Compare(play.Dan, "") == 0 {
		return nil, []string{}
	}
	var dan []string
	dan = strings.Split(play.Dan, ";")

	//检测设胆是否有重复
	sort.Strings(dan) //设胆排序
	a_len := len(dan)
	var flag bool = true
	for i := 0; i < a_len; i++ {
		if (i > 0 && dan[i-1] == dan[i]) || len(dan[i]) == 0 {
			flag = false
			break
		}

		var danIdOne string = fmt.Sprintf("20%s", strings.Replace(dan[i], "-", "", -1))
		if !searchString(matchIdArr, danIdOne) {
			flag = false
			break
		}

	}

	if !flag {
		return errors.New("设胆错误"), dan
	}

	return nil, dan
}

/**
 * @name:根据投注的标示id 获取定义的配置信息
 * @msg:获取的配置信息包含此玩法的全部投注内容，用来验证投注信息是否正确
 * @param lotid int  标示id
 * @return: []string 自定义配置信息
 */
func (play *PlayBase) getBetCode(lotid int) []string {
	var betCode []string
	switch lotid {
	case JcConf.SpfCode: //竞彩足球胜平负
		betCode = JcConf.Spf
	case JcConf.RqspfCode: //竞彩足球让球胜平负
		betCode = JcConf.Rqspf
		break
	case JcConf.BqcCode: //竞彩足球半全场
		betCode = JcConf.Bqc
		break
	case JcConf.BfCode: //竞彩足球比分
		betCode = JcConf.Bf
		break
	case JcConf.ZjqCode: //竞彩足球总进球
		betCode = JcConf.Zjq
		break
	case JcConf.LqSfCode: //篮球胜负
		betCode = JcConf.LqSf
		break
	case JcConf.LqRqSfCode: //篮球让球胜负
		betCode = JcConf.LqRqSf
		break
	case JcConf.LqSfCCode: //篮球胜分差
		betCode = JcConf.LqSfC
		break
	case JcConf.LqDxfCode: //篮球大小分
		betCode = JcConf.LqDxf
		break
	default:

	}
	return betCode
}

/**
 * @name:数组去重
 * @msg:剔除数组中重复的字符串
 * @param a []string 字符串数组
 * @return: ret []string 剔除重复字符串的新数组
 */
func (play *PlayBase) arrayUnique(a []string) (ret []string) {
	a_len := len(a)
	for i := 0; i < a_len; i++ {
		if (i > 0 && a[i-1] == a[i]) || len(a[i]) == 0 {
			continue
		}
		ret = append(ret, a[i])
	}
	return
}

//赛程去重
func (play *PlayBase) arrayUniqueMatch(a []Match) (ret []Match) {
	a_len := len(a)
	for i := 0; i < a_len; i++ {
		if (i > 0 && a[i-1].Issue == a[i].Issue) || len(a[i].Betcode) == 0 {
			continue
		}
		ret = append(ret, a[i])
	}
	return
}

/**
 * @name:混投赛程去重
 * @msg:混合过关赛程 Match数组去重
 * @param a []Match Match数组
 * @return: ret []Match 剔除重复Match的新数组
 */
func (play *PlayBase) arrayUniqueHtMatch(a []Match) (ret []Match) {
	a_len := len(a)
	for i := 0; i < a_len; i++ {
		if (i > 0 && a[i-1].Issue == a[i].Issue && a[i-1].Lotid == a[i].Lotid) || len(a[i].Betcode) == 0 {
			continue
		}
		ret = append(ret, a[i])
	}
	return
}

/**
 * @name:赛程排序
 * @msg:根据定义的赛程唯一标示排序
 * @param a []Match Match数组
 * @return: ret []Match 排序后的Match的新数组
 */
type MatchSlice []Match

func (c MatchSlice) Len() int {
	return len(c)
}
func (c MatchSlice) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
func (c MatchSlice) Less(i, j int) bool {
	if strings.Compare(c[i].Issue, c[j].Issue) == -1 {
		return true
	}
	return false
}

/**
 * @name:整形数组查找
 * @msg:查询数组中是否包含某一个值 -- 整形查找
 * @param []int 整形数组 int 需要查找的数字
 * @return: bool true 包含 不包含
 */
func searchInt(sint []int, index int) bool {
	sort.Ints(sint)
	pos := sort.Search(len(sint), func(i int) bool { return sint[i] >= index })
	if pos < len(sint) && sint[pos] == index {
		return true
	} else {
		return false
	}
}

/**
 * @name:字符串数组查找
 * @msg:查询数组中是否包含某一个值 -- 字符串查找
 * @param []int 字符串数组 string 需要查找的字符串
 * @return: bool true 包含 不包含
 */
func searchString(strArr []string, index string) bool {
	sort.Strings(strArr)
	pos := sort.Search(len(strArr), func(i int) bool { return strArr[i] >= index })
	if pos < len(strArr) && strArr[pos] == index {
		return true
	} else {
		return false
	}
}

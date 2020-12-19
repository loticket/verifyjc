package verifyjc

import (
	"errors"
)

type PlayJingcai struct {
	playBase *PlayBase //验证竞彩足球篮球的投注格式类
}

/**
 * @name:把需要验证的数据传入
 * @param lotnum 投注字符串 money 单倍金额, multiple 倍数, betNum 注数, lotid 玩法id, playtype 过关方式, dan 胆
 * @return: nil
 */
func (play *PlayJingcai) SetTikcet(lotnum string, money int, multiple int, betNum int, lotid int, playtype string, dan string) {
	play.playBase = &PlayBase{
		Lotnum:   lotnum,
		Money:    money,
		Multiple: multiple,
		BetNum:   betNum,
		Lotid:    lotid,
		Playtype: playtype,
		Dan:      dan,
	}
}

/**
 * @name:把需要验证的数据传入
 * @param playbase PlayBase对象
 * @return: error 验证结果信息 []JcTicket 拆票后的票
 */
func (play *PlayJingcai) SetTikcetStruct(playbase *PlayBase) {
	play.playBase = playbase
}

/**
 * @name:验证竞彩足球篮球的投注格式类
 * @msg:验证竞彩足球篮球的投注格式类,并拆票
 * @param nil
 * @return: error 验证结果信息 []JcTicket 拆票后的票
 */

func (play *PlayJingcai) Verification() (error, []JcTicket, int) {

	//[]Match 投注信息解析后的详细信息 []string 赛程ID  []string 投注玩法ID  error 验证投注信息是否错误
	matchArr, matchIdArr, lotidArr, err := play.playBase.MatchRegValidate()
	if err != nil {
		return err, []JcTicket{}, 0
	}
	playtype, single, errs := play.playBase.PlaytypeValidate(lotidArr)
	if err != nil {
		return errs, []JcTicket{}, 0
	}

	errDan, danArr := play.playBase.CheckDan(matchIdArr)
	if errDan != nil {
		return errDan, []JcTicket{}, 0
	}

	var splite *PlaySplite = playSplitePool.Get().(*PlaySplite)

	splite.SetSpliteTicket(matchArr, playtype, play.playBase.Multiple, danArr, play.playBase.Lotid)

	jcTicket, zhu, money, errSplite := splite.GetZuJcTicket()
	if errSplite != nil {
		return errSplite, []JcTicket{}, 0
	}

	if money != play.playBase.Money || zhu != play.playBase.BetNum {
		return errors.New("注数或者金额计算错误"), []JcTicket{}, 0
	}

	defer playSplitePool.Put(splite)

	return nil, jcTicket, single
}

func NewPlayJingcai(lotnum string, money int, multiple int, betNum int, lotid int, playtype string, dan string) *PlayJingcai {
	return &PlayJingcai{
		playBase: &PlayBase{
			Lotnum:   lotnum,
			Money:    money,
			Multiple: multiple,
			BetNum:   betNum,
			Lotid:    lotid,
			Playtype: playtype,
			Dan:      dan,
		},
	}
}

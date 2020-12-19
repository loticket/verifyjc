package verifyjc

//竞彩足球赛程结构
type Match struct {
	Betcode   string   //投注号码
	Matchcode string   //只有投注信息
	Lotid     string   //混合投注使用
	Issue     string   //期号
	Matchnums string   //场次
	Betnum    string   //投注字符串
	Betnumarr []string //投注的数组
	Betlen    int      //投注的次数
	MatchSp   string   //投注本场比赛的sp
}

type JcTicket struct {
	Lotnum   string `json:"lotnum"`   //投注号码
	Issue    string `json:issue`      //期号
	Money    int    `json:"money"`    //钱数
	Multiple int    `json:"multiple"` //倍数
	BetNum   int    `json:"betnum"`   //注数
	Lotid    int    `json:"lotid"`    //彩种编号
	Playtype string `json:"playtype"` //串关方式
	Dan      string `json:"dan"`      //设置的胆
	Lotres   string `json:"lotres"`   //sp值
}

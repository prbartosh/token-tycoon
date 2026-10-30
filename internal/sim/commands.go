package sim

type CommandKind uint8

const (
	CmdBuyBusiness CommandKind = iota + 1
	CmdBuyManager
	CmdBuyUpgrade
	CmdStart
	CmdBoost
	CmdPrimary
	CmdUndo
	CmdBuyRack
	CmdSellRack
	CmdBuyGrid
	CmdBuyPPA
	CmdBuySMR
	CmdBuyOrbital
	CmdSetCloud
	CmdPrestige
	CmdReact
	CmdClaimGrant
	CmdTrain
	CmdCancelTrain
	CmdSetTrainShare
	CmdSetSynthetic
	CmdBuyScraper
	CmdBuyLicense
	CmdSetSegments
	CmdAssignModel
	CmdRetireModel
	CmdHire
	CmdFire
	CmdCounterOffer
	CmdAcceptContract
	CmdDeclineContract
	CmdFunding
	CmdResearch
	CmdAddRule
	CmdDeleteRule
	CmdToggleRule
	CmdMoveRule
	CmdEnterRegion
	CmdCompliance
	CmdRespond
	CmdAGIInvest
)

const (
	Qty1   = 1
	Qty10  = 10
	Qty100 = 100
	QtyMax = -1
)

type Command struct {
	Kind     CommandKind
	Business int
	Upgrade  int
	Gen      int
	Qty      int
	Index    int
	MW       float64
	Params   float64
	Tokens   float64
	Share    float64
	Segments Segments
	On       bool
	Text     string
	Rule     Rule
}

func Buy(business, qty int) Command {
	return Command{Kind: CmdBuyBusiness, Business: business, Qty: qty}
}
func HireManager(business int) Command {
	return Command{Kind: CmdBuyManager, Business: business}
}
func BuyUpgrade(upgrade int) Command {
	return Command{Kind: CmdBuyUpgrade, Upgrade: upgrade, Business: -1}
}
func Start(business int) Command { return Command{Kind: CmdStart, Business: business} }

func Boost(selected int) Command { return Command{Kind: CmdBoost, Business: selected} }
func Primary(business, qty int) Command {
	return Command{Kind: CmdPrimary, Business: business, Qty: qty}
}
func UndoLast() Command { return Command{Kind: CmdUndo, Business: -1} }

func BuyRack(gen, qty int) Command {
	return Command{Kind: CmdBuyRack, Business: -1, Gen: gen, Qty: qty}
}
func SellRack(gen, qty int) Command {
	return Command{Kind: CmdSellRack, Business: -1, Gen: gen, Qty: qty}
}
func BuyGrid(mw float64) Command { return Command{Kind: CmdBuyGrid, Business: -1, MW: mw} }
func BuyPPA(mw float64) Command  { return Command{Kind: CmdBuyPPA, Business: -1, MW: mw} }
func BuySMR() Command            { return Command{Kind: CmdBuySMR, Business: -1} }
func BuyOrbital() Command        { return Command{Kind: CmdBuyOrbital, Business: -1} }
func SetCloud(on bool) Command   { return Command{Kind: CmdSetCloud, Business: -1, On: on} }
func PrestigeReset() Command     { return Command{Kind: CmdPrestige, Business: -1} }
func React(key string) Command   { return Command{Kind: CmdReact, Business: -1, Text: key} }
func ClaimGrant(date string) Command {
	return Command{Kind: CmdClaimGrant, Business: -1, Text: date}
}

func Train(params, tokens, share float64) Command {
	return Command{Kind: CmdTrain, Business: -1, Params: params, Tokens: tokens, Share: share}
}
func CancelTrain(job int) Command { return Command{Kind: CmdCancelTrain, Business: -1, Index: job} }
func SetTrainShare(job int, share float64) Command {
	return Command{Kind: CmdSetTrainShare, Business: -1, Index: job, Share: share}
}
func SetSynthetic(share float64) Command {
	return Command{Kind: CmdSetSynthetic, Business: -1, Share: share}
}
func BuyScraper(qty int) Command { return Command{Kind: CmdBuyScraper, Business: -1, Qty: qty} }
func BuyLicense() Command        { return Command{Kind: CmdBuyLicense, Business: -1} }
func SetSegments(free, pro, enterprise float64) Command {
	return Command{Kind: CmdSetSegments, Business: -1, Segments: Segments{Free: free, Pro: pro, Enterprise: enterprise}}
}
func AssignModel(model int, segment string) Command {
	return Command{Kind: CmdAssignModel, Business: -1, Index: model, Text: segment}
}
func RetireModel(model int) Command { return Command{Kind: CmdRetireModel, Business: -1, Index: model} }

func Hire(class int) Command        { return Command{Kind: CmdHire, Business: -1, Index: class} }
func Fire(i int) Command            { return Command{Kind: CmdFire, Business: -1, Index: i} }
func CounterOffer() Command         { return Command{Kind: CmdCounterOffer, Business: -1} }
func AcceptContract(i int) Command  { return Command{Kind: CmdAcceptContract, Business: -1, Index: i} }
func DeclineContract(i int) Command { return Command{Kind: CmdDeclineContract, Business: -1, Index: i} }
func FundingRound() Command         { return Command{Kind: CmdFunding, Business: -1} }
func Research(node int) Command     { return Command{Kind: CmdResearch, Business: -1, Index: node} }
func AddRule(r Rule) Command        { return Command{Kind: CmdAddRule, Business: -1, Rule: r} }
func DeleteRule(i int) Command      { return Command{Kind: CmdDeleteRule, Business: -1, Index: i} }
func ToggleRule(i int) Command      { return Command{Kind: CmdToggleRule, Business: -1, Index: i} }
func MoveRule(i, delta int) Command {
	return Command{Kind: CmdMoveRule, Business: -1, Index: i, Qty: delta}
}

func EnterRegion(i int) Command   { return Command{Kind: CmdEnterRegion, Business: -1, Index: i} }
func BuyCompliance(i int) Command { return Command{Kind: CmdCompliance, Business: -1, Index: i} }
func Respond(key string) Command  { return Command{Kind: CmdRespond, Business: -1, Text: key} }
func AGIInvest() Command          { return Command{Kind: CmdAGIInvest, Business: -1} }

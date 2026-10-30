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
	Qty      int
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

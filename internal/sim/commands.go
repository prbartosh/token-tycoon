package sim

// CommandKind is the type of a player command.
type CommandKind uint8

const (
	// CmdBuyBusiness buys Qty units of Business (QtyMax = as many as affordable).
	CmdBuyBusiness CommandKind = iota + 1
	// CmdBuyManager hires the manager of Business.
	CmdBuyManager
	// CmdBuyUpgrade buys upgrade Upgrade (index in balance order).
	CmdBuyUpgrade
	// CmdStart starts a manual cycle of Business (no manager, idle).
	CmdStart
	// CmdBoost is a Space press: combo +step (DECISIONS 16). If Business >= 0
	// and it has no manager and is idle, its cycle starts too.
	CmdBoost
	// CmdPrimary is Enter on a business row: buy Qty if affordable, otherwise
	// start its cycle when it has no manager (DECISIONS 16).
	CmdPrimary
	// CmdUndo reverts the last purchase within the undo window.
	CmdUndo
)

// Bulk quantities.
const (
	Qty1   = 1
	Qty10  = 10
	Qty100 = 100
	QtyMax = -1
)

// Command is one player action. Unused fields are ignored.
type Command struct {
	Kind     CommandKind
	Business int
	Upgrade  int
	Qty      int
}

// Convenience constructors for the UI.
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

// Boost is a Space press with the selected business (-1 for none).
func Boost(selected int) Command { return Command{Kind: CmdBoost, Business: selected} }
func Primary(business, qty int) Command {
	return Command{Kind: CmdPrimary, Business: business, Qty: qty}
}
func UndoLast() Command { return Command{Kind: CmdUndo, Business: -1} }

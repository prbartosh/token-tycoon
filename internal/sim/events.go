package sim

type EventKind uint8

const (
	EvCycle EventKind = iota + 1
	EvCrit
	EvPurchase
	EvManager
	EvUpgrade
	EvMilestoneSpeed
	EvMilestoneProfit
	EvMilestoneGlobal
	EvUnlock
	EvUndo
	EvRejected
	EvStart
	EvCombo
	EvComboEnd
	EvRack
	EvRackSold
	EvEnergy
	EvPPAExpired
	EvCloud
	EvStarter
	EvGPUGen
	EvPrestige
	EvCard
	EvCardSet
	EvEventStart
	EvEventEnd
	EvReact
	EvReactMiss
	EvGrant
	EvRivalRelease
	EvSegments
	EvAssign
	EvModelRetired
	EvModelRelease
	EvTrainQueued
	EvTrainCancel
	EvTrainShare
	EvSynthetic
	EvScraper
	EvLicense
	EvScraperIncident
	EvHired
	EvFired
	EvPoach
	EvPoached
	EvCounterOffer
	EvContractOffer
	EvContractAccepted
	EvContractDeclined
	EvContractDone
	EvContractFailed
	EvFunding
	EvResearchStart
	EvResearchDone
	EvRuleAdded
	EvRuleDeleted
	EvRuleToggled
	EvRuleMoved
	EvRuleFired
	EvAutoPrestige
	EvRegion
	EvCompliance
	EvIncident
	EvIncidentOK
	EvIncidentMiss
	EvQuarter
	EvAGIInvest
	EvAGIStage
	EvAGITraining
	EvAGIEnding
	EvSILevel
	EvAchievement
)

type Reject uint8

const (
	RejectNone Reject = iota
	RejectCannotAfford
	RejectLocked
	RejectAlreadyOwned
	RejectNotOwned
	RejectNothingToUndo
	RejectInvalid
	RejectBusy
	RejectNoEvent
	RejectAlreadyClaimed
	RejectBadDate
	RejectNoData
	RejectQueueFull
	RejectTooLarge
	RejectNeedBenchmark
	RejectTeamFull
	RejectNeedResearch
	RejectNoCapital
	RejectNeedRegion
	RejectFinished
)

const (
	EnergyGrid = iota
	EnergyPPA
	EnergySMR
	EnergyOrbital
)

type Event struct {
	Kind     EventKind
	Business int
	Index    int
	Count    int
	Amount   float64
	Value    float64
	Reason   Reject
	Cmd      CommandKind
	Purchase PurchaseKind
}

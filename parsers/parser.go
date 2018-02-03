package parsers

// Parser is the interface that every parser implements
type Parser func(input Input) (ParserResult, Input)

// ParserResult is the interface of the result that every parser returns
type ParserResult interface {
	// Name is the name of the PARSER that yielded this result
	Name() string
	Lines() []int
}

// AllParsers is an array with all of the default parsers
var AllParsers = []Parser{
	ParseKillmail,
	ParseEFT,
	ParseFitting,
	ParseLootHistory,
	ParsePI,
	ParseViewContents,
	ParseMiningLedger,
	ParseWallet,
	ParseSurveyScan,
	ParseContract,
	ParseAssets,
	ParseIndustry,
	ParseCargoScan,
	ParseDScan,
	ParseListing,
}

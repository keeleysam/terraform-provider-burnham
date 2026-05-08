/*
Wordlists for petname generation. Three lists of exactly 64 entries each so the byte-to-index mapping `b % 64` is uniformly distributed (256 / 64 = 4 with no remainder, no modulo bias).

These were composed from scratch for terraform-burnham — short, kid-friendly, lowercase, no profanity, no proper nouns. They're intentionally small (compared to the ~1500-entry upstream `dustinkirkland/golang-petname` corpus) because (a) the byte-uniformity from a power-of-two list size is worth more than larger collision space at the scales this is used, and (b) embedding fewer than 200 short ASCII strings keeps the binary lean.

If a list ever grows to a size that's not a power of two, switch the modulo to a multi-byte uniform draw (see nanoid.go) to avoid bias.
*/

package identifiers

// petnameAdjectives — 64 short, simple adjectives.
var petnameAdjectives = [...]string{
	"amber", "ample", "bold", "brave", "breezy", "bright", "brisk", "calm",
	"clean", "clever", "cool", "cosmic", "cozy", "crisp", "curious", "daring",
	"deep", "dewy", "eager", "easy", "fancy", "fast", "fluffy", "fresh",
	"frosty", "funny", "gentle", "glad", "golden", "grand", "happy", "jolly",
	"kind", "lazy", "lemon", "light", "lively", "lucky", "merry", "mighty",
	"mossy", "neat", "plucky", "polite", "quiet", "rapid", "ready", "royal",
	"salty", "shiny", "silly", "silver", "slow", "snowy", "soft", "spry",
	"sunny", "sweet", "swift", "tidy", "tiny", "vivid", "warm", "witty",
}

// petnameAdverbs — 64 short adverbs that read naturally before an adjective.
var petnameAdverbs = [...]string{
	"abruptly", "almost", "barely", "boldly", "bravely", "briefly", "briskly", "calmly",
	"casually", "cleanly", "cleverly", "deftly", "directly", "eagerly", "easily", "evenly",
	"fairly", "finely", "firmly", "fondly", "freely", "gaily", "gently", "gladly",
	"gracefully", "grandly", "happily", "hastily", "heartily", "humbly", "idly", "jauntily",
	"jovially", "justly", "keenly", "kindly", "lazily", "lightly", "loudly", "mainly",
	"merrily", "mildly", "neatly", "noisily", "openly", "partly", "plainly", "quickly",
	"quietly", "rapidly", "readily", "simply", "slowly", "smartly", "smoothly", "softly",
	"suddenly", "surely", "sweetly", "swiftly", "vastly", "warmly", "wildly", "wisely",
}

// petnameNouns — 64 short, mostly-animal nouns.
var petnameNouns = [...]string{
	"ant", "bear", "beaver", "bird", "bison", "camel", "cat", "cheetah",
	"cow", "crab", "crane", "crow", "deer", "dog", "dolphin", "dove",
	"duck", "eagle", "eel", "elk", "falcon", "ferret", "fox", "frog",
	"gecko", "goat", "goose", "hare", "hawk", "hen", "horse", "koala",
	"lemur", "lion", "lizard", "llama", "mole", "moose", "otter", "owl",
	"ox", "panda", "parrot", "penguin", "pig", "rabbit", "raccoon", "ram",
	"rat", "raven", "salmon", "seal", "shark", "sheep", "snake", "spider",
	"squirrel", "swan", "tiger", "toad", "turkey", "turtle", "walrus", "wolf",
}

// Compile-time invariants: each list is exactly 64 entries (a power of two so byte-mod is unbiased).
var _ [64]string = petnameAdjectives
var _ [64]string = petnameAdverbs
var _ [64]string = petnameNouns

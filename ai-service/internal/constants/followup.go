package constants

import "time"

type FollowupConfig struct {
	Delay time.Duration
	Text  string
}

// FollowupSchedules maps a program name (e.g. "Hairstyling", "Makeup Artistry")
// to a sequence of followups (1-indexed map for stages).
var FollowupSchedules = map[string]map[int]FollowupConfig{
	"Hairstyling": {
		1: {Delay: 1 * time.Minute, Text: "We have flexible schedules for hairstyling: both in-house and hybrid options! Which works better for you?"},
		2: {Delay: 1 * time.Minute, Text: "Quick question - are you familiar with Manitoba Student Aid? Many of our students get their full program covered at 0% interest!"},
		3: {Delay: 84 * time.Hour, Text: "{FirstName}, enrolling at Aveda is a simple process. When are you looking to start school: right away or in the near future?"},
		4: {Delay: 96 * time.Hour, Text: "Not sure what you’re doing for work now, but there are so many career paths when you graduate with us! What are you hoping to do: work at a salon, or maybe for yourself?"},
	},
	"Makeup Artistry": {
		1: {Delay: 24 * time.Hour, Text: "We offer Klarna so you can break down your Makeup course into four interest-free payments. Want the details?"},
		2: {Delay: 24 * time.Hour, Text: "Our Makeup Program is great because you can get certified in only 3 weeks! Want me to send you the full schedule?"},
		3: {Delay: 48 * time.Hour, Text: "One of my favourite parts of our Makeup program is that you get a professional kit! What are you looking forward to most: learning a new skill, or unboxing your new products?"},
		4: {Delay: 48 * time.Hour, Text: "When you finish from our makeup program, there are so many creative paths you can take. What are you hoping to do—work for a makeup brand, or maybe work for yourself?"},
	},
}

package rivernet

import (
	"sort"
	"strings"
)

// displayNames lets a river keep the name people and the department's own
// flood maps use, where the official basin layer calls it something else. The
// other name stays searchable as an alias.
var displayNames = map[string]string{
	"Aruvi Aru": "Malwathu Oya",
}

// siName gives Sinhala names only for rivers whose spelling is certain. A
// river not listed simply has no Sinhala name.
var siName = map[string]string{
	"Kelani Ganga":   "කැලණි ගඟ",
	"Mahaweli Ganga": "මහවැලි ගඟ",
	"Kalu Ganga":     "කළු ගඟ",
	"Walawe Ganga":   "වළවේ ගඟ",
	"Nilwala Ganga":  "නිල්වලා ගඟ",
	"Gin Ganga":      "ගිං ගඟ",
	"Deduru Oya":     "දැදුරු ඔය",
	"Maha Oya":       "මහ ඔය",
	"Kala Oya":       "කලා ඔය",
	"Malwathu Oya":   "මල්වතු ඔය",
	"Gal Oya":        "ගල් ඔය",
	"Kirindi Oya":    "කිරිඳි ඔය",
	"Menik Ganga":    "මැණික් ගඟ",
	"Maduru Oya":     "මාදුරු ඔය",
	"Yan Oya":        "යාන් ඔය",
}

// basinAliases says which river a station's basin name means, where the
// spelling differs from the river's. "Mee Oya" needs care: the official layer
// also has a small, different Mee Oya in the north-east, while the stations'
// Mee Oya at Galgamuwa is the river called Mi Oya.
var basinAliases = map[string]string{
	"mee oya":          "mi oya",
	"aththanagalu oya": "attanagalu oya",
	"attanagala oya":   "attanagalu oya",
	"malwathu oya":     "malwathu oya",
	"benthara ganga":   "bentota ganga",
	"kanagarayan aru":  "kanakarayan aru",
	"uruboku oya":      "urubokka oya",
}

// Fold compares river names: lower case with single spaces.
func Fold(name string) string {
	return strings.Join(strings.Fields(strings.ToLower(name)), " ")
}

// CanonicalBasin folds a station's basin name and resolves its alias. It says
// nothing about whether that river exists: a station from another source may
// name a basin no table knows, and then it is matched by position alone.
func CanonicalBasin(basin string) string {
	folded := Fold(basin)
	if alias, ok := basinAliases[folded]; ok {
		return alias
	}
	return folded
}

// BasinVote says how much of a river system's local catchment lies in a named
// official basin.
type BasinVote struct {
	MainRiv  int64
	BasinFID int64
	Basin    string
	CatchKM2 float64
}

// SystemName is the name a river system takes from the official basins.
type SystemName struct {
	Name     string
	BasinFID int64
}

// ResolveSystemNames names each river system after the official basin it mostly
// lies in. A basin names only its largest system: the small streams beside a
// great river's mouth drain into the same basin polygon but are not that river.
func ResolveSystemNames(votes []BasinVote) map[int64]SystemName {
	total := map[int64]float64{}
	for _, vote := range votes {
		total[vote.MainRiv] += vote.CatchKM2
	}
	// Each system's basin: the one holding most of its catchment.
	best := map[int64]BasinVote{}
	for _, vote := range votes {
		current, ok := best[vote.MainRiv]
		if !ok || vote.CatchKM2 > current.CatchKM2 || (vote.CatchKM2 == current.CatchKM2 && vote.BasinFID < current.BasinFID) {
			best[vote.MainRiv] = vote
		}
	}
	claims := map[int64][]BasinVote{}
	for _, vote := range best {
		if vote.CatchKM2 >= total[vote.MainRiv]/2 {
			claims[vote.BasinFID] = append(claims[vote.BasinFID], vote)
		}
	}
	names := map[int64]SystemName{}
	for fid, list := range claims {
		sort.Slice(list, func(i, j int) bool {
			if list[i].CatchKM2 != list[j].CatchKM2 {
				return list[i].CatchKM2 > list[j].CatchKM2
			}
			return list[i].MainRiv < list[j].MainRiv
		})
		names[list[0].MainRiv] = SystemName{Name: list[0].Basin, BasinFID: fid}
	}
	return names
}

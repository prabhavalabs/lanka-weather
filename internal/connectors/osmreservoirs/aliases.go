package osmreservoirs

// Aliases are other spellings of a reservoir's name, by reservoir identifier,
// where the feed's spelling and OpenStreetMap's differ by more than Key folds.
// Each is the spelling tagged on the OpenStreetMap feature (checked against the
// query's answer on 2026-11-02, with its position and the reservoir's district)
// or, for "Mousakelle" and "Parakrama Samudra", a usual English spelling. An
// alias only widens the name match: the district check still applies.
var Aliases = map[string][]string{
	"ambakolawewa":           {"Abakola Wewa"},
	"badagiriya":             {"Bandagiriya Tank"},
	"bomuruella":             {"Bomburuella Reservoir"},
	"castlereagh":            {"Castlereigh Reservoir"},
	"chandrikawewa":          {"Chandrika Tank"},
	"diyabana":               {"Diyabana Oya Reservoir"},
	"ekgal-oya":              {"Ekagal Oya Tank"},
	"giants-tank":            {"Giant Tank"},
	"hakwatunawa-wawa":       {"Hakwatuna Oya Tank"},
	"kahakurullanpelassa":    {"Kahakurullanpelessa Lake"},
	"kandiyapita-left-bank":  {"Kandiyapita Tank"},
	"kandiyapita-right-bank": {"Kandiyapita Tank"},
	"kekiriobada":            {"Kekirioboda Tank"},
	"magalla":                {"Magalle Wewa"},
	"mahakandarawa":          {"Mahakanadarawa Wewa"},
	"maussakele":             {"Maskeliya Reservoir", "Maussakelle", "Mousakelle"},
	"nagadeepa":              {"Nagadeepa Maha Wewa"},
	"navakiri":               {"Navakiri Aru Tank"},
	"pallan-oya":             {"Jayanthi Wewa"},
	"parakrama-samudraya":    {"Parakrama Samudra"},
	"polgolla":               {"Polgolla Barrage"},
	"rajendrankulam":         {"Rasenthirankulam"},
	"rantambe":               {"Rantembe Dam"},
	"sorabora":               {"Sorabor Wewa"},
	"tampitiya":              {"Tempitiya Wewa"},
	"usgala-siyabalan":       {"Usgala Siyambalangamuwa Tank"},
	"vakaneri":               {"Vahaneri Dam"},
	"victoria":               {"Victoaria Resevoir"},
	"viyathikulam":           {"Viyadikulama Tank"},
}

// Districts are the districts a reservoir lies in, for those the feed gives no
// district for: most of the Mahaweli Authority's. They follow the Mahaweli
// systems the reservoirs belong to, checked against where each landed on
// 2026-11-02; a reservoir on a boundary lists both sides. A reservoir not
// listed here is not mapped.
var Districts = map[string][]string{
	"bowatenna":     {"Matale"},
	"castlereagh":   {"Nuwara Eliya"},
	"chandrikawewa": {"Ratnapura"},
	"dambulu-oya":   {"Matale"},
	"dayaraba":      {"Badulla"},
	"diyabana":      {"Badulla"},
	"hepolaoya":     {"Badulla"},
	"kala-wewa":     {"Anuradhapura"},
	"kaluganga":     {"Matale", "Kandy"},
	"kandalama":     {"Matale"},
	"kotmale":       {"Nuwara Eliya", "Kandy"},
	"loggaloya":     {"Badulla", "Ampara"},
	"maduru-oya":    {"Polonnaruwa", "Ampara", "Batticaloa", "Badulla"},
	"maussakele":    {"Nuwara Eliya", "Ratnapura"},
	"moragahakanda": {"Matale"},
	"polgolla":      {"Kandy"},
	"puhulpola":     {"Badulla", "Nuwara Eliya"},
	"randenigala":   {"Kandy", "Nuwara Eliya", "Badulla"},
	"rantambe":      {"Kandy", "Nuwara Eliya", "Badulla"},
	"samanalawewa":  {"Ratnapura", "Badulla"},
	"udawalawe":     {"Ratnapura", "Monaragala"},
	"ulhitiya":      {"Ampara", "Badulla"},
	"upper-kotmale": {"Nuwara Eliya"},
	"victoria":      {"Kandy", "Nuwara Eliya"},
}

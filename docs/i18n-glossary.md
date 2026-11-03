# Sinhala and Tamil glossary

The official terms the interface uses in Sinhala and Tamil, with where each comes from. The app's
shared words live in `apps/web/src/locales/<lang>/terms.ts` and `format.ts`; feature namespaces
use the same forms. Change a term there only together with this file. See ADR 0016.

Researched on 3 November 2026. The primary sources are the trilingual bulletins the agencies
published that week (PDFs on dmc.gov.lk and irrigation.gov.lk), read from the rendered pages:
text copied out of these PDFs reorders Sinhala vowel signs, and older ones use legacy FM/Bamini
fonts.

Confidence:

- **high**: seen verbatim in an agency document or on the agency's own site;
- **medium**: seen in a news report of an official statement, or built directly from a primary
  phrase;
- **low**: our best term, not found in any source. These need a native reviewer most.

## What the app follows

- **Each agency's own word on its own product.** NBRO's landslide levels are numbered අදියර 1/2/3
  in Sinhala and எச்சரிக்கை மட்டம் 1/2/3 in Tamil, named විමසිලිමත් / අවධාන / ඉවත්වීම and
  அவதானிப்பு / எச்சரிக்கை / வெளியேறுகை, with Amber as ඇම්බර් and செம்மஞ்சள். The Department of
  Meteorology writes Amber as ඇම්බර් and ஆம்பர், and its colour tiers are Red (warning, අනතුරු
  ඇඟවීම, எச்சரிக்கை), Amber (advisory, අවවාදාත්මක, ஆலோசனை) and Green; it has no yellow notice.
- **The Irrigation Department's flood levels.** සුළු ගංවතුර මට්ටම and මහ ගංවතුර මට්ටම (not
  ප්‍රධාන); in Tamil சிறிய வெள்ள மட்டம் and பாரிய வெள்ள மட்டம், built from the department's
  "சிறிய / பாரிய வெள்ளப்பெருக்கு நிலைமை". No Sinhala or Tamil "alert level" label was found;
  අනතුරු ඇඟවීමේ මට්ටම and எச்சரிக்கை மட்டம் are our forms.
- **Directions as words.** The bulletins write directions out in full and never abbreviate them, so
  the app shows නිරිත, தென்மேற்கு and so on, and "from the south-west" as "නිරිත දෙසින්" and
  "தென்மேற்குத் திசையிலிருந்து". West is බටහිර; බස්නාහිර is only the Western Province.
- **Sri Lankan Tamil.** மணித்தியாலம் for hours (not the Indian மணிநேரம்), உத்தியோகபூர்வ for
  official (not அதிகாரப்பூர்வ), and the Sri Lankan month spellings பெப்ரவரி, ஓகஸ்ட், ஒக்டோபர்.
- **Gregorian month names in Sinhala** (නොවැම්බර්), as the Met Department writes them, not the
  lunar names `Intl` gives (ඉල්).
- **Estimates.** No agency publishes wording for "estimate" or "not an official warning"; the app uses
  the general terms ඇස්තමේන්තුව and மதிப்பீடு (section 7), never තක්සේරුව, which means valuation.
- **Names of the agencies.** NBRO became the National Building Research Institute (NBRI) under Act
  No. 20 of 2025 (nbro.gov.lk redirects to nbri.gov.lk). The data sources and the API still say
  NBRO, so the app keeps the acronym NBRO and the name the agency's site header still carries; moving
  to NBRI is a decision for the owner.

## 1. Agencies and official names

| English | Sinhala | Tamil | Source(s) | Confidence |
|---|---|---|---|---|
| Disaster Management Centre (DMC) | ආපදා කළමනාකරණ මධ්‍යස්ථානය | அனர்த்த முகாமைத்துவ நிலையம் | dmc.gov.lk lang=si / lang=ta headers | high |
| District Disaster Management Coordinating Unit | දිස්ත්‍රික් ආපදා කළමනාකරණ මධ්‍යස්ථානය | மாவட்ட அனர்த்த முகாமைத்துவ நிலைய அலகு | DMC DDMCU page (id=94) | high |
| National Emergency Operations Centre | ජාතික හදිසි මෙහෙයුම් මධ්‍යස්ථානය | தேசிய அவசர நடவடிக்கை நிலையம் | DMC menu | high |
| Disaster Management Act | ආපදා කළමනාකරණ පනත | அனர்த்த முகாமைத்துவச் சட்டம் | DMC menu (Act text itself not reachable) | high |
| Department of Meteorology | කාලගුණ විද්‍යා දෙපාර්තමේන්තුව | வளிமண்டலவியல் திணைக்களம் | Met letterhead on every bulletin | high |
| National Meteorological Centre (Forecasting Division) | ජාතික කාලගුණ විද්‍යා මධ්‍යස්ථානය (අනාවැකි අංශය) | தேசிய வளிமண்டலவியல் நிலையம் (முன்னறிவிப்புப் பிரிவு) | Met general forecast 03-11-2026 | high |
| Natural Hazards Early Warning Centre (Met) | ස්වභාවික විපත් පිළිබඳ පූර්ව අනතුරු ඇඟවීමේ මධ්‍යස්ථානය | இயற்கை அனர்த்தம் தொடர்பான முன்னெச்சரிக்கை நிலையம் | Met warnings and advisories (MSA/LRA/LLA series) | high |
| National Building Research Institute (NBRI, formerly NBRO) | ජාතික ගොඩනැගිලි පර්යේෂණ ආයතනය | தேசிய கட்டிட ஆராய்ச்சி நிறுவகம் (site footer: நிறுவனம்) | NBRI landslide early-warning letterhead (LEWM-31-10-2026); nbri.gov.lk | high |
| Former NBRO name (still on the site header) | ජාතික ගොඩනැගිලි පර්යේෂණ සංවිධානය | தேசிய கட்டிட ஆராய்ச்சி அமைப்பு | nbri.gov.lk lang=si/ta header; DMC links | high |
| Irrigation Department | වාරිමාර්ග දෙපාර්තමේන්තුව | நீர்ப்பாசனத் திணைக்களம் | Irrigation letterhead (the website writes நீர்ப்பாசன திணைக்களம்) | high |
| Hydrology and Disaster Management Division | ජලවිද්‍යා හා ආපදා කළමනාකරණ අංශය | நீரியல் மற்றும் அனர்த்த முகாமைத்துவம் (பிரிவு) | Irrigation flood warnings; irrigation.gov.lk menu | high |
| National Tsunami Early Warning Centre | ජාතික සුනාමි පූර්ව අනතුරු ඇඟවීමේ මධ්‍යස්ථානය | சுனாமி முன்னெச்சரிக்கை மையம் | Newsfirst Sinhala; Virakesari 256595 | medium |
| Geological Survey and Mines Bureau (earthquake reports) | භූ විද්‍යා සමීක්ෂණ හා පතල් කාර්යාංශය | புவிச்சரிதவியல் அளவை மற்றும் சுரங்கப் பணியகம் (Virakesari writes "புவியியல் ஆய்வு, சுரங்கப் பணியகம்") | Not verified on gsmb.gov.lk | low |
| Mahaweli Authority of Sri Lanka | ශ්‍රී ලංකා මහවැලි අධිකාරිය | இலங்கை மகாவலி அதிகார சபை | mahaweli.gov.lk has no SI/TA pages; not verified | low |
| Emergency number 117 | හදිසි ඇමතුම් 117 | அவசர தொடர்புகள் 117 | DMC footer (si/ta) | high |
| Emergency alerts (DMC banner) | හදිසි අනතුරු ඇඟවීම් | அவசர எச்சரிக்கைகள் | DMC homepage | high |
| Situation report | තත්ව වාර්තා | நிலவர அறிக்கை | DMC homepage | high |
| Early warning systems | පූර්ව අනතුරු ඇඟවීම් පද්ධති | முன்னெச்சரிக்கை அமைப்புகள் | DMC menu | high |

## 2. NBRI landslide early warning levels

Source: NBRI "Landslide Early Warning" LEWM-31-10-2026-1600-38, a trilingual PDF hosted on DMC. All entries are high confidence and were read from the rendered page.

| English (as printed) | Sinhala (as printed) | Tamil (as printed) | Confidence |
|---|---|---|---|
| Landslide Early Warning (title) | නායයෑම් පිළිබඳ පෙර අනතුරු ඇඟවීම (page 3 says නායයෑම් පිළිබඳ පූර්ව අනතුරු ඇඟවීම්) | மண்சரிவு முன்னெச்சரிக்கை | high |
| Warning level 1: Watch | අදියර 1 : විමසිලිමත් | எச்சரிக்கை மட்டம் 1: அவதானிப்பு | high |
| Colour code: Yellow | වර්ණ කේතය: කහ | வர்ண குறியீடு: மஞ்சள் | high |
| Warning level 2: Alert | අදියර 2 : අවධාන | எச்சரிக்கை மட்டம் 2: எச்சரிக்கை | high |
| Colour code: Amber | වර්ණ කේතය: ඇම්බර් | வர்ண குறியீடு: செம்மஞ்சள் | high |
| Warning level 3: Evacuation | අදියර 3 : ඉවත්වීම | எச்சரிக்கை மட்டம் 3: வெளியேறுகை | high |
| Colour code: Red | වර්ණ කේතය: රතු | வர்ண குறியீடு: சிவப்பு | high |
| L1 action (>75 mm/24 h): "be watchful…" | …අවදානම පිළිබඳව විමසිලිමත් වන්න. | …சாத்தியங்களுக்கான அவதானிப்புடன் இருக்கவும். | high |
| L2 action (>100 mm): "be on alert… ready to evacuate if the need arises" | …අවධානයෙන් සිට අවශ්‍ය වුවහොත් ආරක්ෂාකාරී තැනකට යාමට සූදානම් වන්න. | …எச்சரிக்கையுடன்- தேவையேற்படின் பாதுகாப்பான இடங்களுக்கு வெளியேறும் தயார் நிலையில் இருக்கவும். | high |
| L3 action (>150 mm): "evacuate to a safe location" | …අනතුරු වලින් වැළකීමට ආරක්ෂිත ස්ථානයකට යන්න. | …அபாயங்களை தவிர்ப்பதற்காக பாதுகாப்பான பகுதிகளுக்கு வெளியேறவும். | high |
| Locations for potential risk | නායයෑමේ අවදානම පැවතිය හැකි ප්‍රදේශ | கனதியான அபாயமுள்ள இடங்கள் | high |
| Landslide (noun) | නායයෑම | மண்சரிவு (NBRI). The DMC and NBRI site menus use நிலச்சரிவு. | high |
| Rainfall within past 24 hours | පසු ගිය පැය 24 තුළ වර්ෂාපතනය | கடந்த 24 மணித்தியாலங்களில் மழைவீழ்ச்சி | high |
| Pre-landslide signs | නායයෑම් පූර්ව ලක්ෂණ | மண்சரிவு முன் அறிகுறிகள் | high |

Notes:
- The DMC homepage tile reads "නායයෑම් අනතුරු" in Sinhala and "நிலச்சரிவு எச்சரிக்கை" in Tamil.
- The short Sinhala imperative "ඉවත් වන්න" appears in the brief, but this bulletin does not use it. The bulletin's level name is the noun **ඉවත්වීම**, and its advice sentence ends in "...ආරක්ෂිත ස්ථානයකට යන්න".
- I found no Tamil "முதலாம் நிலை / இரண்டாம் நிலை / மூன்றாம் நிலை" wording in NBRI material.

## 3. Irrigation Department river flood terms

Sources:
- Flood warning, Kalu Ganga, 2020-05-19 (Sinhala).
- Withdrawal of flood warning, Gin Ganga, 2022-07-04.
- Mahaweli flood warning No. 02 (Red), 2026-08-03. The Sinhala is on p1 and the Tamil on p2.
- Kelani flood warning No. 04 (Amber), 2026-08-03.

| English | Sinhala | Tamil | Source(s) | Confidence |
|---|---|---|---|---|
| Flood Warning Message | ගංවතුර අනතුරු ඇඟවීමේ නිවේදනය | வெள்ள எச்சரிக்கை (அறிவித்தல்) | SI: all Irrigation FWs. TA: the Tamil body text says "எச்சரிக்கை விடுக்கப்படுகிறது"; I found no Tamil title. | SI high / TA medium |
| Flood warnings (website menu) | ගංවතුර අවවාද නිවේදන | (menu is English only) | irrigation.gov.lk lang=si | high |
| Withdrawal of flood warning | ගංවතුර අනතුරු ඇඟවීම් ඉවත් කිරීමේ නිවේදනය / …අනතුරු ඇඟවීම ඉවත් කිරීම | வெள்ள எச்சரிக்கை விலக்கிக்கொள்ளப்படுகிறது | SI: Gin Ganga withdrawal 2022. TA: not seen. | SI high / TA low |
| Colour – Amber / Red (FW header) | වර්ණය - ඇම්බර් / වර්ණය - රතු | (no Tamil header) | Kelani No 04; Mahaweli No 02 | high |
| River basin | ද්‍රෝණිය (e.g. කැළණි ගඟ ද්‍රෝණිය) | வடிநிலம் (e.g. களனி கங்கை வடிநிலப் பகுதி) | Kelani No 04 | high |
| Gauging station | ජල මිනුම් ස්ථානය | நதி நீர்மட்ட அளவீடு (நிலையம்). The website says நீரியல் நிலையம். | Kalu Ganga FW 2020; Kelani No 04 TA | high |
| Water level | ජල මට්ටම | நீர்மட்டம் | FWs; irrigation.gov.lk | high |
| Prevailing water level | පවතින ජල මට්ටම | தற்போதைய நீர்மட்டம் | SI: Kalu FW table. TA: derived. | SI high / TA medium |
| Real-time water level | තථ්‍යකාල ජල මට්ටම් | நிகழ்நேர நீர்மட்டம் | irrigation.gov.lk home si/ta | high |
| Normal (gauge status) | සාමාන්‍ය | சாதாரண | Not seen in SI/TA tables; the DMC table is English only | low-medium |
| Alert level | අනතුරු ඇඟවීමේ මට්ටම | எச்சரிக்கை மட்டம் | Not found in an SI/TA gauge table | medium-low |
| Minor flood level | සුළු ගංවතුර මට්ටම | சிறிய வெள்ள மட்டம் | SI: Kalu FW table header. TA: derived from "சிறிய வெள்ளப் பெருக்கு நிலைமை". | SI high / TA medium |
| Major flood level | මහ ගංවතුර මට්ටම | பாரிய வெள்ள மட்டம் | SI: Kalu FW table header. TA: derived from "பாரிய வெள்ளப்பெருக்கு நிலைமை". | SI high / TA medium |
| Minor flood situation | සුළු ගංවතුර තත්ත්වයක් | சிறிய வெள்ளப் பெருக்கு நிலைமை | Kelani No 04 | high |
| Major flood situation | මහ ගංවතුර තත්ත්වයක් | பாரிய வெள்ளப்பெருக்கு நிலைமை | Mahaweli No 02 | high |
| Areas at risk of minor flooding | සුළු ගංවතුර තර්ජනයට ලක්විය හැකි ප්‍රදේශ | — | Kalu FW 2020 | high |
| Low-lying areas | පහත් බිම් ප්‍රදේශ | தாழ்நிலப் பகுதிகள் | Mahaweli No 02 | high |
| Rising (water level) | ඉහළ නගිමින් පවතී | அதிகரித்து வருகிறது / உயர்ந்து வருகிறது | SI: Kalu FW 2020 ("ඉහළ නගිමින් පවතින"). TA: Kelani says "நீர்மட்டங்களில் … அதிகரிப்பு". | SI high / TA medium |
| Falling (water level) | පහළ යමින් පවතී | குறைந்து வருகிறது | SI: Gin withdrawal ("පහල යමින් පවතින", written with ල). TA: not seen. | SI high / TA low |
| Rainfall | වර්ෂාපතනය (also වැසි) | மழைவீழ்ச்சி | All agencies | high |
| Daily rainfall data | දෛනික වර්ෂාපතන දත්ත | தினசரி மழைவீழ்ச்சித் தரவு | irrigation.gov.lk | high |
| Catchment (upper/middle) | ඉහළ / මධ්‍යම ධාරා ප්‍රදේශ | மேல் / மத்திய நீர்ப்பிடிப்புப் பகுதிகள் | Kelani No 04; Mahaweli No 02 | high |

## 4. Reservoirs

| English | Sinhala | Tamil | Source(s) | Confidence |
|---|---|---|---|---|
| Reservoir | ජලාශය | நீர்த்தேக்கம் | Mahaweli FW 02 (කොත්මලේ ජලාශය / கொத்மலை நீர்த்தேக்கம்); irrigation.gov.lk | high |
| Tank (village/irrigation tank) | වැව (e.g. කලාවැව) | குளம் (e.g. கந்தளாய் குளம், உன்னிச்சைக் குளம்) | Hiru, Lankadeepa; Virakesari | medium |
| Spill gates / spillway gates | වාන් දොරටු | வான் கதவுகள் | Mahaweli FW 02 (both languages) | high |
| Opening of spill gates | වාන් දොරටු විවර කිරීම (news: විවෘත කිරීම / විවෘත කෙරේ) | வான் கதவுகள் திறக்கப்படுதல் / திறப்பு | Mahaweli FW 02; Hiru 466161; Virakesari 246161 | high |
| Spilling (overflowing) | වාන් දමයි / වාන් දැමීම | வான்பாய்கின்றன / வான் பாய்தல் | Ada Derana Sinhala 186407 headline; Virakesari 246161 | medium |
| Releasing water | ජලය මුදා හැරේ / ජල ප්‍රමාණයක් නිකුත් කෙරේ | நீர் வெளியேற்றப்படுகிறது / திறந்து விடப்படுகிறது | Hiru; Virakesari | medium |
| cusecs (cubic feet per second) | තත්පරයට ඝන අඩි | விநாடிக்கு … கன அடி (also செக்கனுக்கு) | Hiru 466161; Virakesari 246161 | medium |
| Cubic metres per second | තත්පරයට ඝන මීටර | விநாடிக்கு கன மீற்றர் | Not seen; spelling follows Met's "கிலோ மீற்றர்" | low |
| Capacity (water capacity) | ජල ධාරිතාවය | கொள்ளளவு | SI: irrigation.gov.lk ("ජලාශයන්හි ජල ධාරිතාවය"). TA: not seen. | SI high / TA low-medium |
| Active / available storage | සක්‍රිය ධාරිතාවය | கிடைத்தற்குரிய சேமிப்பு | irrigation.gov.lk home (si/ta) | high |
| Full supply level (FSL) | පූර්ණ සැපයුම් මට්ටම | முழு வழங்கல் மட்டம் | Not found; consider showing "FSL" with a gloss | low |

## 5. Meteorology: products, systems and hazards

Sources:
- General Weather Forecast, 03-11-2026 05:30, in Sinhala and Tamil.
- Strong winds and rough seas warning MSA/26/11/03/01, in Sinhala and Tamil.
- Heavy rain advisory LRA/26/10/25/01, in Sinhala and Tamil.
- Severe lightning advisories LLA/26/10/27/01 and LLA/26/10/29/01.
- Warm weather advisory LHA/26/10/31/01.

All are trilingual Met PDFs on DMC.

| English | Sinhala | Tamil | Source(s) | Confidence |
|---|---|---|---|---|
| Weather forecast | කාලගුණ අනාවැකිය | வானிலை முன்னறிவிப்பு | General forecast | high |
| General weather forecast for Sri Lanka | ශ්‍රී ලංකාව සඳහා සාමාන්‍ය කාලගුණ අනාවැකිය | இலங்கைக்கான பொதுவான வானிலை முன்னறிவிப்பு | General forecast | high |
| Sea area forecast (next 24 h) | ඉදිරි පැය 24 ට දිවයින වටා වන මුහුදු ප්‍රදේශ සඳහා කාලගුණ අනාවැකිය | அடுத்த 24 மணித்தியாலத்துக்கான, நாட்டைச் சூழவுள்ள கடற் பரப்பிற்கான வானிலை முன்னறிவிப்பு | General forecast p2 | high |
| Fishing and naval communities | ධීවර සහ නාවුක ප්‍රජාව | கடலில் பயணம் செய்வோரும் மீனவ சமூகமும் | Sea area forecast | high |
| Multi-day fishing boats | බහුදින ධීවර යාත්‍රා | பலநாள் மீன்பிடிக் கலங்கள் | MSA warning | high |
| Warning (Red tier) | අනතුරු ඇඟවීම (පියවර ගන්න) | எச்சரிக்கை (நடவடிக்கை எடுக்கவும்) | Colour legend, LRA/LLA advisories | high |
| Advisory (Amber tier) | අවවාදාත්මක (සූදානම් වන්න) | ஆலோசனை (தயாராக இருக்கவும்) | Colour legend | high |
| Green tier | කොළ (බලපෑමක් නැත) = no impact | பச்சை: ஆபத்து நீங்கி விட்டது = danger has passed | Colour legend (note that the two languages say different things) | high |
| Colour: Red / Amber / Green | වර්ණය: රතු / ඇම්බර් / කොළ | நிறம்: சிவப்பு / ஆம்பர் / பச்சை | Bulletin headers and legends | high |
| Yellow | කහ | மஞ்சள் | Not a Met notice tier. NBRI uses it. | high (term) |
| Notice / bulletin; bulletin number | නිවේදනය; නිවේදන අංකය | அறிக்கை; அறிக்கை இல | Met headers | high |
| Advisory for heavy rain | තද වැසි පිළිබඳ අවවාදාත්මක නිවේදනය | பலத்த மழைவீழ்ச்சி தொடர்பான ஆலோசனை | LRA/26/10/25/01 | high |
| Advisory for severe lightning | ප්‍රබල අකුණු පිළිබඳව අවවාදාත්මක නිවේදනය | கடும் மின்னல் தாக்கம் தொடர்பான ஆலோசனை | LLA advisories | high |
| Warning for strong winds and rough seas | තද සුළං සහ මුහුද රළුවීම පිළිබඳ අනතුරු ඇඟවීමේ නිවේදනය | கடும் காற்று மற்றும் கொந்தளிப்பான கடல் தொடர்பான எச்சரிக்கை | MSA/26/11/03/01 | high |
| Warm weather advisory | උණුසුම් කාලගුණ තත්ත්වය පිළිබඳ අවවාදාත්මක නිවේදනය | (Tamil version not found) | LHA/26/10/31/01 | SI high |
| Valid for the next 24 hours | ඉදිරි පැය 24 සඳහා වලංගු වන පරිදි | அடுத்த 24 மணித்தியாலங்களுக்கு செல்லுபடியாகும் வகையில் | MSA warning | high |
| Please be prepared! / Please be attentive! | සූදානම් වන්න! / අවධානයෙන් සිටින්න! | தயவுசெய்து தயாராக இருக்கவும்! / தயவுசெய்து அவதானமாக இருக்கவும்! | LRA; MSA | high |
| Cyclone (generic) | සුළි කුණාටුව | சூறாவளி | MSA SI/TA (the TA table header reads "சூறாவளி தளம்பல் வகை") | high |
| Cyclonic storm (CS) | සුළි කුණාටුව | புயல் | MSA warning and sea forecast | high |
| Severe cyclonic storm (SCS) | ප්‍රබල සුළි කුණාටුව | பாரிய புயல் | Sea forecast p2 (SI), MSA (TA) | high |
| Deep depression | ගැඹුරු පීඩන අවපාතය | ஆழமான தாழமுக்கம் | MSA warning | high |
| Depression | පීඩන අවපාතය | தாழமுக்கம் | MSA warning; LRA | high |
| Low pressure area | අඩු පීඩන කලාපය | குறைந்த அழுத்தப் பிரதேசம் | MSA table; LRA | high |
| Well-marked low pressure area | (not seen) | நன்கமைந்த குறைந்த அழுத்தப் பிரதேசம் | MSA TA table | TA high |
| Strong winds | තද සුළං | பலத்த காற்று (also கடும் காற்று in titles) | Forecast; MSA | high |
| Temporary strong winds (in thundershowers) | තාවකාලික තද සුළං | தற்காலிகமாக வீசக் கூடிய பலத்த காற்று | General forecast | high |
| Rough seas / sea becomes rough | මුහුද රළු වීම / රළු විය හැක | கொந்தளிப்பான கடல் / கொந்தளிப்பாகக் காணப்படும் | MSA; sea forecast | high |
| Very rough | ඉතා රළු | மிகவும் கொந்தளிப்பாக | Sea forecast | high |
| Moderate (sea) | මද වශයෙන් රළු | மிதமான அலையுடன் | Sea forecast | high |
| Sea condition / state of sea | මුහුදේ ස්වභාවය | கடல் நிலை | Sea forecast | high |
| Condition of rain | වැසි තත්ත්වය | மழை நிலைமை | Sea forecast | high |
| Winds (heading) | සුළඟ | காற்று | Sea forecast | high |
| Wind speed | සුළඟේ වේගය | காற்றின் வேகம் | Sea forecast | high |
| km/h | පැ.කි.මී. (පැයට කිලෝමීටර) | மணித்தியாலத்துக்கு … கிலோ மீற்றர் | Sea forecast; MSA | high |
| Wind direction | සුළං දිශාව | காற்றின் திசை | Not seen as a heading. The phrase "…திசையிலிருந்து வீசக் கூடும்" is attested. | medium |
| Heavy rain | තද වැසි | பலத்த மழை | Rainfall legend | high |
| Showers | වැසි | மழை | Forecast | high |
| Thundershowers | ගිගුරුම් සහිත වැසි | இடியுடன் கூடிய மழை | Forecast | high |
| Lightning | අකුණු | மின்னல் (மின்னல் தாக்கம்) | Forecast; LLA | high |
| Misty conditions | මීදුම් සහිත තත්ත්වයක් | பனிமூட்டமான நிலை | Forecast | high |
| Rainfall amount (mm), categories | වර්ෂාපතනය (මි.මී.): සිහින් වැසි 0–12.5, සිහින් හෝ මද වැසි 12.5–25, මද වැසි 25–50, තරමක් තද වැසි 50–100, තද වැසි 100–150, ඉතා තද වැසි 150 හෝ ඊට වැඩි | மழைவீழ்ச்சி அளவு (மி.மீ): சிறிதளவு 0–12.5, சிறிதளவிலிருந்து மிதமானது வரை 12.5–25, மிதமானது 25–50, ஓரளவு பலத்த மழை 50–100, பலத்த மழை 100–150, மிகப் பலத்த மழை 150 க்கும் அதிகமானது | Footer of every Met forecast | high |
| Risk level legend on maps (High/Moderate/Low/No risk) | (English only on maps) | (English only on maps) | Advisory maps | — |
| Heat index levels: Normal / Caution / Extreme caution / Danger | සාමාන්‍යයි / අවධානය / වැඩි අවධානය / අනතුරුදායකයි | (Tamil version not found) | LHA/26/10/31/01 p2 | SI high |
| Heat index | උණුසුම් දර්ශකය | வெப்பச் சுட்டெண் | SI: LHA. TA: not seen. | SI high / TA low |

## 6. Earthquake, tsunami and other observations

| English | Sinhala | Tamil | Source(s) | Confidence |
|---|---|---|---|---|
| Earthquake | භූකම්පනය / භූ කම්පනය | நிலநடுக்கம் (also பூகம்பம்) | Ada Derana Sinhala 223534; Virakesari 256595, 221279 | medium |
| Magnitude | ප්‍රබලත්වය (e.g. "රික්ටර් මාපකයේ 7.4ක ප්‍රබලත්වයෙන්") | ரிக்டர் அளவு (e.g. "7.7 ரிக்டர் அளவில்") | same | medium |
| Tsunami | සුනාමි | சுனாமி | same | medium-high |
| No tsunami threat to Sri Lanka | ශ්‍රී ලංකාවට සුනාමි අවදානමක් නැත | இலங்கைக்கு சுனாமி அச்சுறுத்தல் இல்லை | Newsfirst Sinhala 2023-12-30 headline; Virakesari 256595 | medium |
| Temperature (maximum) | උෂ්ණත්වය (උපරිම උෂ්ණත්වය) | வெப்பநிலை (உயர் வெப்பநிலை) | SI: LHA p2. TA: not seen. | SI high / TA medium |
| Humidity (relative) | ආර්ද්‍රතාවය (සාපේක්ෂ ආර්ද්‍රතාවය) | ஈரப்பதன் (சாரீரப்பதன் for relative humidity) | SI: LHA p2 ("වායුගෝලීය ආර්ද්‍රතාවය"). TA: not seen. | SI high / TA low |
| Air quality / AQI | වායු ගුණත්වය / වායු ගුණත්ව දර්ශකය | காற்றின் தரம் / காற்றுத் தரச் சுட்டெண் | No SI/TA primary found (NBRI AQI reports are English) | low-medium |
| Rain gauge | වර්ෂාමානය | மழைமானி | Not found in sources | low |
| Weather station | කාලගුණ මධ්‍යස්ථානය | வானிலை அவதான நிலையம் | Not found. Irrigation uses ජල විද්‍යා ස්ථාන / நீரியல் நிலையங்கள் for hydrological stations. | low |
| Hydrological station | ජල විද්‍යා ස්ථානය | நீரியல் நிலையம் | irrigation.gov.lk | high |

## 7. Estimates, official status and disclaimers

None of the agencies publishes this wording. These are standard terms.

| English | Sinhala | Tamil | Source(s) | Confidence |
|---|---|---|---|---|
| Estimate (noun) | ඇස්තමේන්තුව | மதிப்பீடு | General usage | medium |
| Estimated | ඇස්තමේන්තුගත / ඇස්තමේන්තු කළ | மதிப்பிடப்பட்ட | General usage. Avoid තක්සේරුව, which means valuation or assessment. | medium |
| Official | නිල | உத்தியோகபூர்வ | General usage (government Tamil uses உத்தியோகபூர்வ, not அதிகாரப்பூர்வ) | medium-high |
| This is not an official warning. | මෙය නිල අනතුරු ඇඟවීමක් නොවේ. | இது உத்தியோகபூர்வ எச்சரிக்கை அல்ல. | Pattern follows Met LHA "මෙය … නොවේ" / EN "This is not the forecast…" | medium |
| Follow instructions from DMC / NBRI | ආපදා කළමනාකරණ මධ්‍යස්ථානය සහ ජාතික ගොඩනැගිලි පර්යේෂණ ආයතනය ලබා දෙන උපදෙස් අනුගමනය කරන්න. | அனர்த்த முகாமைத்துவ நிலையம் மற்றும் தேசிய கட்டிட ஆராய்ச்சி நிறுவகம் என்பவற்றின் அறிவுறுத்தல்களை பின்பற்றவும். | Adapted from the NBRI LEWM p3 note | high (pattern) |

## 8. Administrative divisions

| English | Sinhala | Tamil | Source(s) | Confidence |
|---|---|---|---|---|
| Province | පළාත | மாகாணம் | Met forecasts | high |
| District | දිස්ත්‍රික්කය | மாவட்டம் | Met forecasts; DMC DDMCU page | high |
| District Secretariat | දිස්ත්‍රික් ලේකම් කාර්යාලය | மாவட்ட செயலகம் | DMC DDMCU page | high |
| Divisional Secretariat (DS) division | ප්‍රාදේශීය ලේකම් කොට්ඨාසය. NBRI and Irrigation bulletins spell it කොට්ඨාශය. | பிரதேச செயலகப் பிரிவு | NBRI LEWM; Irrigation FWs (SI/TA) | high |
| Grama Niladhari (GN) division | ග්‍රාම නිලධාරී වසම | கிராம உத்தியோகத்தர் பிரிவு (older form: கிராம சேவகர் பிரிவு) | Standard government usage; not confirmed in a primary source here | medium |

### Provinces

Sinhala names are from Met forecasts and advisories (high). Tamil names are from Met forecasts (high) unless marked otherwise.

| English | Sinhala | Tamil | Confidence |
|---|---|---|---|
| Western Province | බස්නාහිර පළාත | மேல் மாகாணம் | high |
| Central Province | මධ්‍යම පළාත | மத்திய மாகாணம் | high |
| Southern Province | දකුණු පළාත | தென் மாகாணம் | high |
| Uva Province | ඌව පළාත | ஊவா மாகாணம் | high |
| Sabaragamuwa Province | සබරගමුව පළාත | சப்ரகமுவ மாகாணம் | high |
| North Western Province | වයඹ පළාත | வடமேல் மாகாணம் | high |
| North Central Province | උතුරු මැද පළාත (Met writes උතුරු-මැද) | வட மத்திய மாகாணம் | SI high / TA medium-high (irrigation.gov.lk region "வட மத்திய") |
| Northern Province | උතුරු පළාත | வட மாகாணம் (Met list reads "வடக்கு … மாகாணங்கள்") | SI high / TA medium-high |
| Eastern Province | නැගෙනහිර පළාත | கிழக்கு மாகாணம் | SI high / TA medium-high |

### Districts

All 25 are from the DMC District units page (si/ta), confidence high. Where the Met Department spells a name differently, both spellings are given.

| Province | English | Sinhala | Tamil |
|---|---|---|---|
| Western | Colombo | කොළඹ | கொழும்பு |
| Western | Gampaha | ගම්පහ | கம்பஹா |
| Western | Kalutara | කළුතර | களுத்துறை |
| Central | Kandy | මහනුවර | கண்டி |
| Central | Matale | මාතලේ | மாத்தளை |
| Central | Nuwara Eliya | නුවරඑළිය (Met) / නුවරඑලිය (DMC) | நுவரெலியா |
| Southern | Galle | ගාල්ල | காலி |
| Southern | Matara | මාතර | மாத்தறை |
| Southern | Hambantota | හම්බන්තොට | அம்பாந்தோட்டை |
| Northern | Jaffna | යාපනය | யாழ்ப்பாணம் |
| Northern | Kilinochchi | කිලිනොච්චි | கிளிநொச்சி |
| Northern | Mannar | මන්නාරම | மன்னார் |
| Northern | Vavuniya | වවුනියාව (Met) / වවුනියා (DMC) | வவுனியா |
| Northern | Mullaitivu | මුලතිව් | முல்லைத்தீவு |
| Eastern | Batticaloa | මඩකලපුව | மட்டக்களப்பு |
| Eastern | Ampara | අම්පාර | அம்பாறை |
| Eastern | Trincomalee | ත්‍රිකුණාමලය | திருகோணமலை |
| North Western | Kurunegala | කුරුණෑගල | குருநாகல் |
| North Western | Puttalam | පුත්තලම | புத்தளம் |
| North Central | Anuradhapura | අනුරාධපුර (DMC; Met writes අනුරාධපුර දිස්ත්‍රික්කය) | அனுராதபுரம் |
| North Central | Polonnaruwa | පොළොන්නරුව | பொலன்னறுவை |
| Uva | Badulla | බදුල්ල | பதுளை |
| Uva | Monaragala | මොණරාගල | மொணராகலை |
| Sabaragamuwa | Ratnapura | රත්නපුර (Met) / රත්නපුරය (DMC) | இரத்தினபுரி |
| Sabaragamuwa | Kegalle | කෑගල්ල | கேகாலை |

## 9. Compass points and monsoons

The Met Department writes directions out in full in the body text. I found **no Sinhala or Tamil abbreviations** (equivalents of N/NE/E) in any bulletin. The cyclone maps keep English labels. If space is short, either keep N/NE/E or use the full word. Do not invent abbreviations.

| English | Sinhala (noun / "from the X") | Tamil (noun / "from the X") | Source(s) | Confidence |
|---|---|---|---|---|
| North | උතුර / උතුරු දෙසින් | வடக்கு / வடக்குத் திசையிலிருந்து | MSA SI ("උතුරට"); MSA TA ("வடக்கு - வடமேற்கு திசையில்") | high |
| North-east | ඊසාන / ඊසාන දෙසින් (ඊසාන දිග) | வடகிழக்கு / வடகிழக்குத் திசையிலிருந்து | DMC news ("ඊසාන දිග මෝසම්"); Ada Derana Tamil ("வடகிழக்குத் திசையில் இருந்து") | high |
| East | නැගෙනහිර | கிழக்கு | Sea forecast ("නැගෙනහිර දේශාංශ" / "கிழக்கு நெட்டாங்கு") | high |
| South-east | ගිනිකොන / ගිනිකොන දෙසින් | தென்கிழக்கு / தென்கிழக்குத் திசையிலிருந்து | Not seen in the sampled bulletins | medium |
| South | දකුණ / දකුණු දෙසින් | தெற்கு / தெற்குத் திசையிலிருந்து | Sea forecast ("නිරිත හෝ දකුණු දෙසින්" / "தென்மேற்கு அல்லது தெற்குத் திசையிலிருந்து") | high |
| South-west | නිරිත / නිරිත දෙසින් (නිරිත දිග) | தென்மேற்கு / தென்மேற்குத் திசையிலிருந்து | Sea forecast | high |
| West | බටහිර / බටහිර දෙසින් (**not** බස්නාහිර) | மேற்கு / மேற்குத் திசையிலிருந்து | Sea forecast ("බටහිර-මධ්‍යම බොක්ක" / "மேற்கு - மத்திய வங்காள விரிகுடா") | high |
| North-west | වයඹ / වයඹ දෙසට | வடமேற்கு / வடமேற்குத் திசையில் | MSA SI ("වයඹ දෙසට"); TA ("வடமேற்குத் திசையில்") | high |
| Latitude N / Longitude E | උතුරු අක්ෂාංශ / නැගෙනහිර දේශාංශ | வட அகலாங்கு / கிழக்கு நெட்டாங்கு | Sea forecast | high |
| Southwest monsoon | නිරිත දිග මෝසම | தென்மேற்கு பருவப்பெயர்ச்சி | Lankadeepa / Divaina quoting the Met Department; Ada Derana Tamil 163025 ("தென்மேற்கு பருவப் பெயர்ச்சி நிலைமை") | medium-high |
| Northeast monsoon | ඊසාන දිග මෝසම | வடகிழக்கு பருவப்பெயர்ச்சி | DMC news title (SI); TA by analogy with the southwest form | SI high / TA medium |
| Inter-monsoon | අන්තර් මෝසම් | பருவப்பெயர்ச்சிக்கு இடைப்பட்ட காலம் | Not verified | low |
| Bay of Bengal | බෙංගාල බොක්ක | வங்காள விரிகுடா | MSA | high |

## 10. Relative time

These are not agency terms. The values below were checked against ICU/CLDR in Node 24 (ICU 78.3). Sri Lankan agencies write **පැය** in Sinhala and **மணித்தியாலம்** in Tamil.

| English | Sinhala | Tamil (Sri Lankan style) | ICU/CLDR output | Confidence |
|---|---|---|---|---|
| just now | දැන් (or මොහොතකට පෙර) | இப்போது (or சற்று முன்) | si "දැන්", ta "இப்போது" | high |
| 12 min ago | මිනිත්තු 12කට පෙර | 12 நிமிடங்களுக்கு முன் | identical | high |
| 1 h ago | පැය 1කට පෙර | 1 மணித்தியாலத்துக்கு முன் | ta "1 மணிநேரம் முன்" (Indian form) | high (SI) / medium (TA) |
| 3 h ago | පැය 3කට පෙර | 3 மணித்தியாலங்களுக்கு முன் | ta "3 மணிநேரம் முன்" | high (SI) / medium (TA) |
| yesterday | ඊයේ | நேற்று | identical | high |
| 2 days ago | දින 2කට පෙර (numeric:auto gives පෙරේදා) | 2 நாட்களுக்கு முன் (numeric:auto gives நேற்று முன்தினம்) | identical | high |
| Short form (ICU) | same as the long form | "12 நிமி. முன்", "3 ம. முன்" (terse; avoid in safety text) | — | — |

Date and time conventions in the Met bulletins:
- Sinhala months are transliterated, e.g. **2026 නොවැම්බර් මස 03 දින පෙ.ව. 05.30**. AM is පෙ.ව. and PM is ප.ව.
- Tamil dates read **2026 நவம்பர் 03ஆம் திகதி அதிகாலை 05.30 மணிக்கு**. AM is மு.ப and PM is பி.ப.
- ICU's `si` locale formats month names as the traditional names (ඉල්), so either pass month names explicitly or format the month as a number.

## Sources consulted

- https://www.dmc.gov.lk/index.php?lang=si and https://www.dmc.gov.lk/index.php?lang=ta: DMC names in SI/TA; 117; names of the Met Dept, NBRO and Irrigation Dept; menu terms; "ගංගා ජල මට්ටම" / "நதி நீரின் நிலை".
- https://www.dmc.gov.lk/index.php?option=com_content&view=article&id=94&Itemid=271&lang=si and the `lang=ta` version: all 25 district names in SI/TA, plus "දිස්ත්‍රික් ලේකම් කාර්යාලය" / "மாவட்ட செயலகம்".
- https://www.dmc.gov.lk/index.php?option=com_dmcreports&view=reports&report_type_id=2|3|5|6|1 : DMC report archive. These list pages gave the PDF links below.
- https://www.dmc.gov.lk/images/dmcreports/Landslide_EW_Report_at_1600hrs_on_2026__1793443439.pdf: NBRI landslide level names, colours and action wording in SI/TA; NBRI letterhead names.
- https://www.dmc.gov.lk/images/dmcreports/Warning_for_strong_winds_and_rough_seas_at_1030hrs_on_03__1793687070.pdf (SI) and …1793687398.pdf (TA): cyclone, depression and low-pressure terms; Natural Hazards EWC name; "ඇම්බර්" / "ஆம்பர்".
- https://www.dmc.gov.lk/images/dmcreports/Weather_Report_at_0530hrs_on_2026__1793664845.pdf (SI) and …1793669554.pdf (TA): forecast, sea area, wind, sea state, rainfall categories, province names.
- https://www.dmc.gov.lk/images/dmcreports/Advisory_for_Heavy_Rain_at_0530_hrs_on_2026__1792894123.pdf (SI) and Advisory_for_Heavy_Rain_(Tamil_Language)__1792894181.pdf: the Red/Amber/Green tier legend in both languages.
- https://www.dmc.gov.lk/images/dmcreports/Advisory_for_Severe_Lightning_at_1200hrs_on_29__1793258154.pdf and …1130hrs_on_27__1793085463.pdf (TA): lightning advisory titles.
- https://www.dmc.gov.lk/images/dmcreports/Warm_Weather_Advisory_at_1600hrs_on_31__1793443588.pdf: heat index levels in SI; temperature and humidity terms.
- https://www.dmc.gov.lk/images/dmcreports/FW_KaluGanga_Minor_2020__1589875140.pdf: SI සුළු ගංවතුර මට්ටම / මහ ගංවතුර මට්ටම / ජල මිනුම් ස්ථානය; "වර්ණය - ඇම්බර්".
- https://www.dmc.gov.lk/images/dmcreports/Withdrawal_of_flood_warning_Gin_Basin__1656909212.pdf: SI withdrawal wording; "පහල යමින්".
- https://www.dmc.gov.lk/images/dmcreports/Water_level_&_Rainfall_2026__1793689764.pdf: the river gauge table is English only (Alert / Minor Flood / Major Flood Level, Normal, Rising or Falling).
- https://www.irrigation.gov.lk/web/index.php?option=com_content&view=category&layout=blog&id=35&Itemid=206 : list of flood warning PDFs.
- https://www.irrigation.gov.lk/web/images/Flood-Warning-Message/Flood_Warning__for_Mahaweli_River_-_No_02_-_20260803_Red.pdf: SI/TA for spill gates, reservoir, major flood situation, low-lying areas and DS divisions.
- https://www.irrigation.gov.lk/web/images/Flood-Warning-Message/Flood_warning_for_Kelani_Basin_-_No_04_-_20260803.pdf: SI/TA for minor flood situation and river gauge terms.
- https://www.irrigation.gov.lk/web/index.php?lang=si and the `lang=ta` version: department name; ජල ධාරිතාවය / සක්‍රිය ධාරිතාවය; நிகழ்நேர நீர்மட்டம் / கிடைத்தற்குரிய சேமிப்பு / நீரியல் நிலையங்கள். Plain curl is blocked by mod_security; a browser User-Agent is needed.
- https://nbri.gov.lk/index.php?lang=si and the `lang=ta` version: confirmed the NBRI rename (Act No. 20 of 2025). The site header still carries the old NBRO names. The warnings page currently shows "No Landslide Early Warnings…".
- https://www.virakesari.lk/article/246161: Tamil வான்பாய்கின்றன, வான் கதவுகள் திறக்கப்பட்டுள்ளன, விநாடிக்கு … கன அடி நீர் வெளியேற்றப்படுகிறது.
- https://hirunews.lk/466161/spill-gates-of-deduru-oya-and-inginimitiya-reservoirs-opened: Sinhala වාන් දොරටු … විවෘත කර; තත්පරයට ඝන අඩි.
- https://www.virakesari.lk/article/256595: Tamil ரிக்டர் அளவில் … நிலநடுக்கம்; சுனாமி அச்சுறுத்தல் இல்லை; சுனாமி முன்னெச்சரிக்கை மையம்.
- https://sinhala.adaderana.lk/news/223534: Sinhala රික්ටර් මාපකයේ … ප්‍රබලත්වයෙන් යුත් … භූකම්පනය; සුනාමි.
- https://adaderanatamil.lk/news/163025: Tamil "தென்மேற்கு பருவப் பெயர்ச்சி".
- https://www.virakesari.lk/article/237843: Tamil "வடகிழக்குத் திசையில் இருந்து காற்று வீசும்".
- Lankadeepa and Divaina search results quoting the Met Department: Sinhala "නිරිත දිග මෝසම් තත්වය".

Unreachable or not useful:
- https://www.meteo.gov.lk: connection timed out from curl, and WebFetch got ECONNREFUSED. The Met Department's own SI/TA web pages were therefore not checked. All Met terms come from Met PDFs mirrored on DMC.
- https://www.documents.gov.lk: TLS handshake failure, and a 404 for the NBRI bill PDF. The trilingual Disaster Management Act No. 13 of 2005 text and the NBRI Act text were not checked.
- https://mahaweli.gov.lk: the /si/ and /ta/ paths returned stub pages, so the Mahaweli SI/TA names are unverified.
- Web search coverage of Sinhala and Tamil text was poor. Several searches returned only irrelevant or Indian results.

import type { Translation } from "@/lib/i18n"
import type { install as en } from "../en/install"

/** Button names in quotes stay in English: that is what the phone itself shows. */
export const install: Translation<typeof en> = {
  open: "යෙදුම ස්ථාපනය කරන්න",
  openHint: "Lanka Weather ඔබගේ දුරකථනයට හෝ පරිගණකයට එක් කරන්න",
  title: "Lanka Weather ස්ථාපනය කරන්න",
  description: "එය ඔබගේ දුරකථනයට හෝ පරිගණකයට එක් කර වෙනත් ඕනෑම යෙදුමක් මෙන් විවෘත කරන්න.",
  alertsReason:
    "iPhone සහ iPad මත, Lanka Weather ඔබගේ Home Screen එකට එක් කළ පසු පමණක් ඇඟවීම් ක්‍රියා කරයි.",
  installNow: "ස්ථාපනය කරන්න",
  installNowHint: "ඔබගේ බ්‍රව්සරයට එක් තට්ටුවකින් එය ස්ථාපනය කළ හැක.",
  orSteps: "නැතහොත් මෙම පියවර අනුගමනය කරන්න:",
  stepsFor: "{device} · {browser}",
  nearest: "මෙය කුමන බ්‍රව්සරයදැයි අපට විශ්වාස නැත. මෙම පියවර එවැනි බොහෝ බ්‍රව්සරවල ක්‍රියා කරයි.",
  otherDevice: "වෙනත් දුරකථනයක් හෝ බ්‍රව්සරයක් භාවිත කරනවාද?",
  device: "උපාංගය",
  browser: "බ්‍රව්සරය",
  devices: {
    iphone: "iPhone හෝ iPad",
    android: "Android",
    computer: "පරිගණකය",
  },
  browsers: {
    safari: "Safari",
    chrome: "Chrome",
    edge: "Edge",
    firefox: "Firefox",
    samsung: "Samsung Internet",
  },
  step: "පියවර {number}",
  steps: {
    "iphone-safari": {
      s1: "Share බොත්තම තට්ටු කරන්න: ඉහළට ඊතලයක් සහිත කොටුවකි. එය නොපෙනේ නම්, පළමුව ⋯ තට්ටු කරන්න.",
      s2: "පහළට අනුචලනය කර “Add to Home Screen” තට්ටු කරන්න.",
      s3: "ඉහළ කෙළවරේ ඇති “Add” තට්ටු කරන්න.",
      s4: "ඔබගේ Home Screen එකේ ඇති නව අයිකනයෙන් Lanka Weather විවෘත කරන්න. ඇඟවීම් එතැනින් සක්‍රිය කරන්න.",
    },
    "iphone-chrome": {
      s1: "ලිපින තීරුවේ ඇති Share බොත්තම තට්ටු කරන්න. නැතහොත් ⋯ තට්ටු කර, පසුව “Share” තට්ටු කරන්න.",
      s2: "“Add to Home Screen” තට්ටු කරන්න. එය නොපෙනේ නම් පහළට අනුචලනය කරන්න.",
      s3: "ඉහළ කෙළවරේ ඇති “Add” තට්ටු කරන්න.",
      s4: "ඔබගේ Home Screen එකේ ඇති නව අයිකනයෙන් Lanka Weather විවෘත කරන්න. ඇඟවීම් එතැනින් සක්‍රිය කරන්න.",
    },
    "iphone-edge": {
      s1: "තිරයේ පහළ ඇති ⋯ තට්ටු කර, පසුව “Share” තට්ටු කරන්න.",
      s2: "“Add to Home Screen” තට්ටු කරන්න. එය නොපෙනේ නම් පහළට අනුචලනය කරන්න.",
      s3: "ඉහළ කෙළවරේ ඇති “Add” තට්ටු කරන්න.",
      s4: "ඔබගේ Home Screen එකේ ඇති නව අයිකනයෙන් Lanka Weather විවෘත කරන්න. ඇඟවීම් එතැනින් සක්‍රිය කරන්න.",
    },
    "iphone-firefox": {
      s1: "මෙනු බොත්තම ☰ තට්ටු කර, පසුව “Share” තට්ටු කරන්න.",
      s2: "“Add to Home Screen” තට්ටු කරන්න. එය නොපෙනේ නම් පහළට අනුචලනය කරන්න.",
      s3: "ඉහළ කෙළවරේ ඇති “Add” තට්ටු කරන්න.",
      s4: "ඔබගේ Home Screen එකේ ඇති නව අයිකනයෙන් Lanka Weather විවෘත කරන්න. ඇඟවීම් එතැනින් සක්‍රිය කරන්න.",
    },
    "android-chrome": {
      s1: "ඉහළ දකුණේ ඇති ⋮ තට්ටු කරන්න.",
      s2: "“Add to Home screen” හෝ “Install app” තට්ටු කරන්න.",
      s3: "“Install” තට්ටු කරන්න.",
      s4: "ඔබගේ Home screen එකෙන් හෝ යෙදුම් ලැයිස්තුවෙන් Lanka Weather විවෘත කරන්න.",
    },
    "android-samsung": {
      s1: "පහළ දකුණේ ඇති මෙනු බොත්තම ☰ තට්ටු කරන්න.",
      s2: "“Add page to” තට්ටු කර, පසුව “Home screen” තට්ටු කරන්න.",
      s3: "“Add” තට්ටු කරන්න. ලිපින තීරුවේ බාගැනීම් අයිකනයක් පෙනේ නම්, ඒ වෙනුවට එය තට්ටු කර “Install” තට්ටු කළ හැක.",
      s4: "ඔබගේ Home screen එකෙන් හෝ යෙදුම් ලැයිස්තුවෙන් Lanka Weather විවෘත කරන්න.",
    },
    "android-edge": {
      s1: "තිරයේ පහළ ඇති ⋯ තට්ටු කරන්න.",
      s2: "“Add to phone” හෝ “Install app” තට්ටු කරන්න.",
      s3: "“Install” තට්ටු කරන්න.",
      s4: "ඔබගේ Home screen එකෙන් හෝ යෙදුම් ලැයිස්තුවෙන් Lanka Weather විවෘත කරන්න.",
    },
    "android-firefox": {
      s1: "ලිපින තීරුව අසල ඇති ⋮ තට්ටු කරන්න.",
      s2: "“Add app to Home screen” තට්ටු කරන්න. එය නොපෙනේ නම්, “More” යටතේ බලන්න.",
      s3: "“Add” තට්ටු කරන්න.",
      s4: "ඔබගේ Home screen එකෙන් හෝ යෙදුම් ලැයිස්තුවෙන් Lanka Weather විවෘත කරන්න.",
    },
    "computer-chrome": {
      s1: "ලිපින තීරුවේ දකුණු කෙළවරේ ඇති ස්ථාපන අයිකනය ක්ලික් කරන්න: පහළට ඊතලයක් සහිත තිරයකි.",
      s2: "අයිකනයක් නැද්ද? ⋮ ක්ලික් කර, “Cast, save, and share”, පසුව “Install Lanka Weather” ක්ලික් කරන්න.",
      s3: "“Install” ක්ලික් කරන්න. Lanka Weather තමන්ගේම කවුළුවක විවෘත වේ.",
    },
    "computer-edge": {
      s1: "ලිපින තීරුවේ ඇති “App available” අයිකනය ක්ලික් කරන්න.",
      s2: "අයිකනයක් නැද්ද? ⋯ ක්ලික් කර, “Apps”, පසුව “Install Lanka Weather” ක්ලික් කරන්න.",
      s3: "“Install” ක්ලික් කරන්න. Lanka Weather තමන්ගේම කවුළුවක විවෘත වේ.",
    },
    "computer-safari": {
      s1: "තිරයේ ඉහළ මෙනු තීරුවේ ඇති “File” ක්ලික් කරන්න.",
      s2: "“Add to Dock” ක්ලික් කරන්න.",
      s3: "“Add” ක්ලික් කරන්න. Lanka Weather දැන් ඔබගේ Dock එකේ ඇත.",
    },
    "computer-firefox": {
      s1: "පරිගණකයක ඇති Firefox මගින් තවමත් වෙබ් යෙදුම් ස්ථාපනය කළ නොහැක.",
      s2: "මෙම පිටුව Chrome හෝ Edge තුළ, නැතහොත් Mac එකක Safari තුළ විවෘත කර එහි ස්ථාපනය කරන්න.",
      s3: "නැතහොත් එය පිටු සලකුණක් ලෙස තබා ගන්න: Ctrl+D ඔබන්න (Mac එකක ⌘D).",
    },
  },
  notes: {
    iosVersion: "ඇඟවීම් සඳහා iOS හෝ iPadOS 16.4 හෝ ඊට පසු සංස්කරණයක් අවශ්‍ය වේ.",
    macVersion: "macOS Sonoma (14) හෝ ඊට පසු සංස්කරණයක් අවශ්‍ය වේ.",
  },
  inApp: {
    title: "මෙම පිටුව ඔබගේ බ්‍රව්සරයේ විවෘත කරන්න",
    body: "{app} මෙම පිටුව තමන්ගේම බ්‍රව්සරයක විවෘත කළ අතර, එයට යෙදුම් ස්ථාපනය කළ නොහැක.",
    bodyUnknown:
      "මෙම යෙදුම මෙම පිටුව තමන්ගේම බ්‍රව්සරයක විවෘත කළ අතර, එයට යෙදුම් ස්ථාපනය කළ නොහැක.",
    ios: "⋯ හෝ Share බොත්තම තට්ටු කර, පසුව “Open in Safari” තට්ටු කරන්න. නැතහොත් සබැඳිය පිටපත් කර Safari තුළ අලවන්න.",
    android:
      "⋮ තට්ටු කර, පසුව “Open in Chrome” හෝ “Open in browser” තට්ටු කරන්න. නැතහොත් සබැඳිය පිටපත් කර Chrome තුළ අලවන්න.",
    copy: "සබැඳිය පිටපත් කරන්න",
    copied: "සබැඳිය පිටපත් විය. එය ඔබගේ බ්‍රව්සරයේ අලවන්න.",
    copyFailed: "පිටපත් කළ නොහැකි විය. ලිපිනය පිටපත් කිරීමට එය ඔබාගෙන සිටින්න.",
    openChrome: "Chrome තුළ විවෘත කරන්න",
  },
  installed: {
    title: "Lanka Weather ස්ථාපනය කර ඇත",
    body: "ඔබ භාවිත කරන්නේ ස්ථාපිත යෙදුමයි. එය සම්පූර්ණ තිරයෙන් විවෘත වන අතර, ඔබ නොබැඳි විට සඳහා අවසන් දත්ත තබා ගනී.",
    alerts: "ඔබගේ ප්‍රදේශයේ අනතුරු ඇඟවීම් ගැන දැනගැනීමට ඇඟවීම් සක්‍රිය කරන්න.",
    alertSettings: "ඇඟවීම් සැකසුම්",
  },
  why: {
    title: "ස්ථාපනය කරන්නේ ඇයි?",
    alerts: "ඔබ අසල අනතුරු ඇඟවීම් පිළිබඳ ඇඟවීම්, iPhone මතද",
    fullScreen: "යෙදුමක් මෙන් සම්පූර්ණ තිරයෙන් විවෘත වේ",
    offline: "අවසන් වරට සුරැකි දත්ත සමඟ අන්තර්ජාලය නැතිවද ක්‍රියා කරයි",
    small: "කුඩා සහ නොමිලේ, යෙදුම් වෙළඳසැලක් අවශ්‍ය නැත",
  },
  prompt: {
    title: "Lanka Weather ස්ථාපනය කරනවාද?",
    body: "ඇඟවීම් ලබා ගෙන අන්තර්ජාලය නැතිවද සිතියම භාවිත කරන්න.",
    show: "කෙසේදැයි පෙන්වන්න",
    later: "දැන් නොවේ",
  },
  thanks: {
    title: "Lanka Weather ස්ථාපනය කළාට ස්තූතියි",
    body: "එය ඔබගේ Home Screen එකෙන් හෝ යෙදුම් ලැයිස්තුවෙන් විවෘත කරන්න.",
  },
  close: "වසන්න",
}

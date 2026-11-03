import type { Translation } from "@/lib/i18n"
import type { pwa as en } from "../en/pwa"

export const pwa: Translation<typeof en> = {
  update: {
    title: "புதிய பதிப்பு தயார்",
    description: "அதைப் பயன்படுத்த மீண்டும் ஏற்றவும். நீங்கள் திறந்திருப்பவை எதுவும் இழக்கப்படாது.",
    reload: "மீண்டும் ஏற்று",
    later: "பின்னர்",
  },
  offline: {
    title: "நீங்கள் இணைய இணைப்பில் இல்லை",
    body: "வரைபடம் கடைசியாகச் சேமித்த தரவைக் காட்டுகிறது. ஒவ்வொரு அடுக்கும் அது எவ்வளவு பழையது எனக் காட்டுகிறது.",
    back: "மீண்டும் இணைப்பில். வரைபடம் புதுப்பிக்கப்படுகிறது.",
  },
  ready: "Lanka Weather இப்போது இணையம் இல்லாமலும் இயங்கும்.",
  push: {
    genericTitle: "Lanka Weather",
    genericBody: "புதிய தகவல் உள்ளது. அதைப் பார்க்க வரைபடத்தைத் திறக்கவும்.",
  },
}

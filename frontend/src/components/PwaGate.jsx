const isPwa =
  window.matchMedia('(display-mode: standalone)').matches ||
  window.navigator.standalone === true

const ua = navigator.userAgent
const isIOS     = /iphone|ipad|ipod/i.test(ua)
const isAndroid = /android/i.test(ua)

const IOS_STEPS = [
  { icon: '📤', title: 'Tap the Share button', desc: 'Find it at the bottom of Safari.' },
  { icon: '📲', title: 'Tap "Add to Home Screen"', desc: 'Scroll down in the share sheet if needed.' },
  { icon: '🚀', title: 'Open Pager and revisit this link', desc: 'Launch the app from your home screen.' },
]

const ANDROID_STEPS = [
  { icon: '⋮',  title: 'Tap the browser menu', desc: 'Three dots in the top-right corner of Chrome.' },
  { icon: '📲', title: 'Tap "Add to Home Screen" or "Install app"', desc: 'The label depends on your browser version.' },
  { icon: '🚀', title: 'Open Pager and revisit this link', desc: 'Launch the app from your home screen.' },
]

const DESKTOP_STEPS = [
  { icon: '📱', title: 'Open this link on your phone', desc: 'Use your iPhone or Android device.' },
  { icon: '📲', title: 'Install Pager to your home screen', desc: 'Follow the on-screen instructions for your OS.' },
  { icon: '🔔', title: 'Notifications work on mobile', desc: 'Pager delivers updates to your lock screen.' },
]

function Step({ number, icon, title, desc }) {
  return (
    <div className="flex items-start gap-4 bg-surface border border-border rounded-xl px-4 py-3">
      <span className="text-2xl leading-none mt-0.5">{icon}</span>
      <div className="flex-1 min-w-0">
        <p className="text-sm font-semibold text-[#f0f0f0] leading-snug">
          <span className="text-brand mr-1">{number}.</span>{title}
        </p>
        <p className="text-xs text-[#888] mt-0.5 leading-snug">{desc}</p>
      </div>
    </div>
  )
}

export default function PwaGate({ children }) {
  if (isPwa) return children

  const steps = isIOS ? IOS_STEPS : isAndroid ? ANDROID_STEPS : DESKTOP_STEPS
  const heading = isIOS
    ? 'Install on iPhone or iPad'
    : isAndroid
      ? 'Install on Android'
      : 'Designed for mobile'

  return (
    <div className="min-h-dvh bg-[#0f0f0f] text-[#f0f0f0] flex flex-col px-5 py-10 gap-6 font-sans max-w-lg mx-auto">
      <h1 className="text-2xl font-bold tracking-tight">
        Pa<span className="text-brand">g</span>er
      </h1>

      <div className="flex flex-col gap-1">
        <h2 className="text-xl font-semibold leading-tight">{heading}</h2>
        <p className="text-[#aaa] text-sm leading-relaxed">
          Open this link inside the Pager app for the best experience.
        </p>
      </div>

      <div className="flex flex-col gap-3">
        {steps.map((step, i) => (
          <Step key={i} number={i + 1} {...step} />
        ))}
      </div>
    </div>
  )
}

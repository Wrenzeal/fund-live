import { ImageResponse } from 'next/og'

export const alt = 'FundLive — Fund estimates, disclosures and research'
export const size = { width: 1200, height: 630 }
export const contentType = 'image/png'

export default function OpenGraphImage() {
  return new ImageResponse(
    <div style={{ width: '100%', height: '100%', display: 'flex', flexDirection: 'column', justifyContent: 'space-between', background: '#f6f1e8', padding: '70px 80px', color: '#18212f', fontFamily: 'sans-serif' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 20 }}>
        <div style={{ display: 'flex', width: 58, height: 58, borderRadius: 14, background: '#256f8f', color: 'white', alignItems: 'center', justifyContent: 'center', fontSize: 36 }}>F</div>
        <span style={{ fontSize: 42, fontWeight: 700 }}>FundLive</span>
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 18 }}>
        <div style={{ display: 'flex', fontSize: 64, fontWeight: 700, lineHeight: 1.15 }}>Fund data, with context.</div>
        <div style={{ display: 'flex', fontSize: 30, color: '#58606c' }}>Estimates · Public disclosures · Market research</div>
      </div>
      <div style={{ display: 'flex', justifyContent: 'space-between', borderTop: '2px solid #d7cdbd', paddingTop: 24, fontSize: 24, color: '#58606c' }}>
        <span>Sources and limitations made visible</span><span>fund.wrenzeal.top</span>
      </div>
    </div>,
    size,
  )
}

import http from 'node:http'

const port = Number(process.env.SEO_FIXTURE_PORT || 18084)
function payload(id) {
  const now = new Date()
  const day = now.toISOString().slice(0, 10)
  return {
    fund: { id, name: id === '320007' ? '公开目录测试基金' : '服务端观察测试基金', type: 'hybrid', manager: '测试经理', company: '测试公司', nav: '1.1', scale: '0', updated_at: now.toISOString() },
    observation: {
      checked_at: now.toISOString(), window_days: 45, latest_holding_period: '2026-09-30', holding_age_days: 33,
      disclosed_weight_percent: '42.62', holdings_count: 10, holdings_source: 'direct_holdings', holdings_status: 'available',
      notices: [{ id: 'notice-1', title: '可追溯的基金公开披露', scope: 'fund', published_at: day, source_name: '测试公开来源', source_url: 'https://example.com/fund/disclosure' }],
      sources: [{ code: 'fund_notices', status: 'ok', checked_at: now.toISOString() }, { code: 'holding_notices', status: 'ok', checked_at: now.toISOString() }],
      notes: ['持仓来自定期披露，不代表当前全部持仓。'],
    },
  }
}

http.createServer((request, response) => {
  response.setHeader('Content-Type', 'application/json')
  if (request.url.startsWith('/api/v1/announcements')) {
    if (request.headers.cookie || request.headers.authorization) { response.writeHead(500).end('{}'); return }
    const now = new Date().toISOString()
    const item = { id: 'release-one', title: '公开产品更新测试', summary: '匿名服务端产品公告摘要', content: '可在禁用 JavaScript 后直接阅读的完整公告正文。', source_type: 'manual', source_ref: '', published_at: now, created_at: now, updated_at: now }
    if (request.url === '/api/v1/announcements') { response.end(JSON.stringify({ success: true, data: [item] })); return }
    if (request.url === '/api/v1/announcements/release-one') { response.end(JSON.stringify({ success: true, data: item })); return }
    if (request.url === '/api/v1/announcements/missing-one') { response.writeHead(404).end(JSON.stringify({ success: false, error: { code: 'ANNOUNCEMENT_NOT_FOUND' } })); return }
    response.writeHead(503).end(JSON.stringify({ success: false })); return
  }
  const match = request.url.match(/^\/api\/v1\/fund\/(\d{6})\/observation$/)
  if (!match) { response.end(JSON.stringify({ success: true, data: null })); return }
  const id = match[1]
  if (request.headers.cookie || request.headers.authorization) {
    response.writeHead(500).end(JSON.stringify({ success: false, error: { code: 'PRIVATE_HEADERS_FORWARDED' } })); return
  }
  if (id === '000002') {
    response.writeHead(404).end(JSON.stringify({ success: false, error: { code: 'FUND_NOT_FOUND' } })); return
  }
  if (id === '000008') {
    response.writeHead(404).end(JSON.stringify({ success: false, error: { code: 'ROUTE_MISSING' } })); return
  }
  if (!['000001', '000003', '000004', '000006', '000007', '320007'].includes(id)) {
    response.writeHead(503).end(JSON.stringify({ success: false, error: { code: 'OBSERVATION_UNAVAILABLE' } })); return
  }
  const data = payload(id)
  if (id === '000003') data.observation.sources[1].status = 'partial'
  if (id === '000004') data.fund.id = '111111'
  if (id === '000006') data.observation.checked_at = new Date(Date.now() - 48 * 3600000).toISOString()
  if (id === '000007') {
    data.fund.name = '测试基金</script><script id="injected-script">bad()</script>'
    data.observation.notices[0].source_url = 'javascript:alert(1)'
  }
  response.end(JSON.stringify({ success: true, data }))
}).listen(port, '127.0.0.1', () => console.log(`SEO fixture listening on ${port}`))

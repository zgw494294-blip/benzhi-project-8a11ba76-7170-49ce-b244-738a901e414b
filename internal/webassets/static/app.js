const $ = (selector, root = document) => root.querySelector(selector);
const $$ = (selector, root = document) => [...root.querySelectorAll(selector)];
const state = { current: null, detail: null, counter: 1 };

function key(prefix) { return `${prefix}-${Date.now()}-${state.counter++}`; }
function values(text) { return text.split(',').map(v => v.trim()).filter(Boolean); }
function iso(value) { return new Date(value).toISOString(); }
function escapeHTML(value = '') { return String(value).replace(/[&<>'"]/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;',"'":'&#39;','"':'&quot;'}[c])); }
function toast(message, error = false) { const el = $('#toast'); el.textContent = message; el.className = error ? 'show error' : 'show'; setTimeout(() => el.className = '', 3000); }
async function api(path, options = {}) {
  const response = await fetch(path, { ...options, headers: options.body ? {'Content-Type':'application/json', ...(options.headers||{})} : options.headers });
  const data = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(data.message || `请求失败 ${response.status}`);
  return data;
}
function meta(actor = '档案整理员') { return { expectedRevision: state.detail.case.revision, idempotencyKey: key('ui'), actor }; }
async function refreshList() {
  const cases = await api('/api/v1/cases');
  $('#case-list').innerHTML = cases.map(c => `<button data-id="${escapeHTML(c.id)}" class="${c.id===state.current?'active':''}"><strong>${escapeHTML(c.title)}</strong><br><span class="muted">${escapeHTML(c.state)} · r${c.revision}</span></button>`).join('') || '<span class="muted">尚无案卷</span>';
  $$('#case-list button').forEach(button => button.onclick = () => selectCase(button.dataset.id));
}
async function selectCase(id) {
  state.current = id; state.detail = await api(`/api/v1/cases/${encodeURIComponent(id)}`); render(); await refreshList();
}
function card(lines, actions = '') { return `<div class="card">${lines.join('<br>')}${actions}</div>`; }
function render() {
  const d = state.detail, c = d.case; $('#empty').hidden = true; $('#detail').hidden = false;
  $('#case-code').textContent = `${c.id} · ${c.intervieweeCode}`; $('#case-title').textContent = c.title;
  $('#case-state').textContent = c.state; $('#case-revision').textContent = `revision ${c.revision}`;
  $('#blockers').innerHTML = d.blockingItems.length ? d.blockingItems.map(x => `<div class="blocker"><strong>${escapeHTML(x.code)}</strong> ${escapeHTML(x.message)} ${escapeHTML(x.evidence||'')}</div>`).join('') : '<div class="blocker" style="border-color:var(--green);background:#e3eee8">当前没有冻结阻断项。</div>';
  const coverage = d.consentCoverage || {};
  $('#consents').innerHTML = `<div class="blocker">覆盖状态：${coverage.covered ? '已完整覆盖' : '存在缺口'}${coverage.warning ? `；${escapeHTML(coverage.warning)}` : ''}<br>${(coverage.items || []).map(x => `${escapeHTML(x.dimension)}「${escapeHTML(x.value)}」：${x.covered ? '已覆盖' : escapeHTML(x.reasonCode)}${x.evidenceRef?.length ? `（${escapeHTML(x.evidenceRef.join('、'))}）` : ''}`).join('<br>')}</div>` + c.consents.map(x => card([`<code>${escapeHTML(x.id)}</code> ${escapeHTML(x.evidenceRef)}`,`受众：${escapeHTML(x.allowedAudiences.join('、'))}；用途：${escapeHTML(x.allowedPurposes.join('、'))}`, x.withdrawnAt ? '已撤回' : `有效至 ${new Date(x.expiresAt).toLocaleString()}`], x.withdrawnAt ? '' : `<br><button data-withdraw="${escapeHTML(x.id)}">撤回同意</button>`)).join('');
  $('#transcripts').innerHTML = c.transcripts.map(x => card([`<code>${escapeHTML(x.id)}</code> · ${x.segments.length} 个片段`,`摘要 ${escapeHTML(x.contentDigest)}`])).join('');
  $('#diff').textContent = d.latestDifferences.length ? JSON.stringify({summary:d.transcriptImpact,items:d.latestDifferences}, null, 2) : '当前版本没有可显示的差异。';
  const latest = c.transcripts[c.transcripts.length-1]; if (latest) { $('#finding-form [name=transcriptVersionId]').value = latest.id; $('#finding-form [name=segmentId]').value = latest.segments[0]?.id || ''; $('#transcript-form [name=baseVersionId]').value = latest.id; }
  $('#findings').innerHTML = c.findings.map(x => card([`<code>${escapeHTML(x.id)}</code> ${escapeHTML(x.category)} · ${escapeHTML(x.status)}`,escapeHTML(x.riskReason),`方案：${escapeHTML(x.redactionProposal||'未提供')}`])).join('');
  $('#objections').innerHTML = c.objections.map(x => card([`<code>${escapeHTML(x.id)}</code> ${escapeHTML(x.reason)}`,x.closedAt ? `已由 ${escapeHTML(x.closedBy)} 关闭，证据 ${escapeHTML(x.resolutionEvidenceRef)}` : '尚未闭环'], x.closedAt ? '' : `<br><button data-close-objection="${escapeHTML(x.id)}">引用最新转写并关闭</button>`)).join('');
  $('#manifest').textContent = c.frozenManifest ? JSON.stringify(c.frozenManifest, null, 2) : (d.freezePreview?.manifestDigest ? `候选摘要：${d.freezePreview.manifestDigest}\n规则版本：${d.freezePreview.ruleVersion}\n证据数：${d.freezePreview.evidence.length}` : '尚未生成候选摘要。');
  $('#credentials').innerHTML = c.credentials.map(x => card([`<code>${escapeHTML(x.id)}</code> ${escapeHTML(x.status)}`,`边界：${escapeHTML(x.audienceScope.join('、'))} / ${escapeHTML(x.purposeScope.join('、'))}`,`校验摘要 ${escapeHTML(x.verificationDigest)}`], `<br><button data-verify="${escapeHTML(x.id)}">立即验真</button>`)).join('');
  $('#audits').innerHTML = d.auditTimeline.map(x => `<li><strong>#${x.sequence} ${escapeHTML(x.action)}</strong> · ${escapeHTML(x.outcome)}<br><span class="muted">${new Date(x.occurredAt).toLocaleString()} · ${escapeHTML(x.actor)} · ${escapeHTML(x.digest)}</span></li>`).join('');
  bindDynamic();
}
function bindDynamic() {
  $$('[data-withdraw]').forEach(b => b.onclick = () => mutate(`/api/v1/cases/${state.current}/consents/${b.dataset.withdraw}/withdraw`, {...meta('档案整理员')}, '同意已撤回'));
  $$('[data-close-objection]').forEach(b => b.onclick = () => { const latest = state.detail.case.transcripts.at(-1); mutate(`/api/v1/cases/${state.current}/objections/${b.dataset.closeObjection}/close`, {...meta('伦理复核员'), resolutionEvidenceRef:latest.id}, '异议已闭环'); });
  $$('[data-verify]').forEach(b => b.onclick = async () => { try { const v=await api(`/api/v1/cases/${state.current}/credentials/${b.dataset.verify}/verify`); toast(`${v.valid?'有效':'无效'}：${v.reason}`, !v.valid); } catch(e){toast(e.message,true);} });
}
async function mutate(path, body, message) { try { await api(path,{method:'POST',body:JSON.stringify(body)}); await selectCase(state.current); toast(message); } catch(e){ toast(e.message,true); await selectCase(state.current); } }

$('#new-case').onclick = () => $('#create-dialog').showModal(); $('[data-close]').onclick = () => $('#create-dialog').close();
$('#create-form').onsubmit = async event => { event.preventDefault(); const f=new FormData(event.target); try { const c=await api('/api/v1/cases',{method:'POST',body:JSON.stringify({idempotencyKey:key('create'),actor:'档案整理员',title:f.get('title'),collectionUnit:f.get('collectionUnit'),intervieweeCode:f.get('intervieweeCode'),sourceRef:f.get('sourceRef'),requestedScope:{audiences:values(f.get('audiences')),purposes:values(f.get('purposes'))}})}); $('#create-dialog').close(); await selectCase(c.id); toast('案卷草稿已创建'); } catch(e){toast(e.message,true);} };
$('#consent-form').onsubmit = event => { event.preventDefault(); const f=new FormData(event.target); mutate(`/api/v1/cases/${state.current}/consents`,{...meta(),evidenceRef:f.get('evidenceRef'),allowedAudiences:values(f.get('audiences')),allowedPurposes:values(f.get('purposes')),effectiveAt:iso(f.get('effectiveAt')),expiresAt:iso(f.get('expiresAt'))},'同意范围核验通过'); };
$('#transcript-form').onsubmit = event => { event.preventDefault(); const f=new FormData(event.target); let segments; try{segments=JSON.parse(f.get('segments'));}catch{toast('片段 JSON 格式错误',true);return;} mutate(`/api/v1/cases/${state.current}/transcripts`,{...meta(f.get('actor')),baseVersionId:f.get('baseVersionId'),segments},'转写版本已提交'); };
$('#finding-form').onsubmit = event => { event.preventDefault(); const f=new FormData(event.target); const raw=String(f.get('batchItems')||'').trim(); if(raw){ let items; try{items=JSON.parse(raw);}catch{toast('批量条目 JSON 格式错误',true);return;} mutate(`/api/v1/cases/${state.current}/findings`,{...meta(),items},'敏感项批次已登记'); } else mutate(`/api/v1/cases/${state.current}/findings`,{...meta(),transcriptVersionId:f.get('transcriptVersionId'),segmentId:f.get('segmentId'),category:f.get('category'),riskReason:f.get('riskReason'),redactionProposal:f.get('redactionProposal')},'敏感项已登记'); };
$('#objection-form').onsubmit = event => { event.preventDefault(); const f=new FormData(event.target); mutate(`/api/v1/cases/${state.current}/objections`,{...meta(f.get('actor')),findingId:f.get('findingId'),reason:f.get('reason')},'异议已退回'); };
$('#freeze').onclick = () => mutate(`/api/v1/cases/${state.current}/freeze`,{...meta('档案整理员'),confirmedManifestDigest:state.detail.freezePreview.manifestDigest},'候选版本已冻结');
$('#authorize').onclick = () => { const c=state.detail.case; mutate(`/api/v1/cases/${state.current}/authorize`,{...meta('公开负责人'),scope:c.requestedScope},'公开授权凭据已签发'); };
$$('.steps button').forEach((button,i) => button.onclick = () => { $$('.steps button').forEach(x=>x.classList.remove('selected')); button.classList.add('selected'); $$('.panel').forEach(p=>p.hidden=p.dataset.name!==button.dataset.panel); });
$$('.steps button')[0].click();
const now=new Date(), later=new Date(Date.now()+365*86400000); $('#consent-form [name=effectiveAt]').value=now.toISOString().slice(0,16); $('#consent-form [name=expiresAt]').value=later.toISOString().slice(0,16);
api('/healthz').then(() => {$('#health').textContent='本地服务就绪'; refreshList();}).catch(e => {$('#health').textContent='服务不可用';toast(e.message,true);});

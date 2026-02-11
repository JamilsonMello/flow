const API_BASE = 'http://localhost:8585/api';
let allFlows = [];
let currentFlowId = null;
let currentFlow = null;
let statusFilter = '';
let lastFlowHash = '';

let flowPage = 1;
let flowLimit = 20;
let flowTotalPages = 1;
let flowLoading = false;

let timelinePage = 1;
let timelineLimit = 50;
let timelineTotalPages = 1;
let timelineLoading = false;

let allExpanded = false;

// ───── Init ─────
document.addEventListener('DOMContentLoaded', () => {
    loadStats();
    refreshFlows();

    document.getElementById('flowList').addEventListener('scroll', handleFlowScroll);
    document.getElementById('timelineContainer').addEventListener('scroll', handleTimelineScroll);

    // Auto-refresh every 5s
    setInterval(() => {
        loadStats();
        const list = document.getElementById('flowList');
        if (list.scrollTop < 50 && !flowLoading) {
            fetchFlows(false, false);
        }
    }, 5000);
});

// ───── Stats ─────
async function loadStats() {
    try {
        const res = await fetch(`${API_BASE}/stats`);
        const s = await res.json();
        document.getElementById('statTotal').textContent = s.total_flows || 0;
        document.getElementById('statActive').textContent = s.active_flows || 0;
        document.getElementById('statFinished').textContent = s.finished_flows || 0;
        document.getElementById('statInterrupted').textContent = s.interrupted_flows || 0;
    } catch (e) { console.error('Stats error:', e); }
}

// ───── Status Filter ─────
function setStatusFilter(status, btn) {
    statusFilter = status;
    document.querySelectorAll('.filter-btn').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    flowPage = 1;
    refreshFlows();
}

// ───── Flow Scroll ─────
function handleFlowScroll(e) {
    const el = e.target;
    if (el.scrollHeight - el.scrollTop - el.clientHeight < 50) {
        if (!flowLoading && flowPage < flowTotalPages) {
            flowPage++;
            fetchFlows(true, true);
        }
    }
}

function handleTimelineScroll(e) {
    const el = e.target;
    if (el.scrollHeight - el.scrollTop - el.clientHeight < 100) {
        if (!timelineLoading && timelinePage < timelineTotalPages) {
            timelinePage++;
            loadMoreTimeline();
        }
    }
}

// ───── Fetch Flows ─────
async function refreshFlows() {
    flowPage = 1;
    const list = document.getElementById('flowList');
    list.innerHTML = '<div style="padding:20px;text-align:center;color:var(--text-muted)">Loading...</div>';
    await fetchFlows(true, false);
}

async function fetchFlows(forceRender = false, append = false) {
    if (flowLoading && append) return;
    flowLoading = true;

    try {
        const search = document.getElementById('searchInput').value;
        let url = `${API_BASE}/flows?page=${flowPage}&limit=${flowLimit}`;
        if (statusFilter) url += `&status=${statusFilter}`;
        if (search) url += `&search=${encodeURIComponent(search)}`;

        const res = await fetch(url);
        const response = await res.json();
        const newFlows = response.data || [];
        flowTotalPages = response.meta.pages;

        if (append) {
            allFlows = [...allFlows, ...newFlows];
            renderFlowList(true);
        } else {
            // Only re-render if data actually changed (prevents flickering)
            const newHash = newFlows.map(f => `${f.id}:${f.status}:${f.point_count}:${f.assertion_count}`).join('|');
            if (newHash !== lastFlowHash || allFlows.length !== newFlows.length) {
                allFlows = newFlows;
                lastFlowHash = newHash;
                renderFlowList(false);
            }
        }
    } catch (e) { console.error(e); }
    finally { flowLoading = false; }
}

function filterFlows() {
    flowPage = 1;
    fetchFlows(true, false);
}

// ───── Render Flow List ─────
function renderFlowList(append = false) {
    const list = document.getElementById('flowList');

    if (!append) {
        list.innerHTML = '';
        if (allFlows.length === 0) {
            list.innerHTML = '<div style="padding:20px;text-align:center;color:var(--text-muted)">No flows found</div>';
            return;
        }
    }

    let flowsToRender = append ? allFlows.slice((flowPage - 1) * flowLimit) : allFlows;
    flowsToRender.forEach(f => renderFlowItem(f, list));
}

function renderFlowItem(f, container) {
    const el = document.createElement('div');
    el.className = `flow-item ${currentFlowId === f.id ? 'active' : ''}`;
    el.onclick = () => selectFlow(f, el);

    let statusClass = 'status-finished';
    if (f.status === 'ACTIVE') statusClass = 'status-active';
    else if (f.status === 'INTERRUPTED') statusClass = 'status-interrupted';

    const date = new Date(f.created_at);
    const dateStr = date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
    const timeStr = date.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });

    const serviceBadge = f.service ? `<span class="service-badge">${f.service}</span>` : '';
    const identBadge = f.identifier ? `<span class="flow-id" title="${f.identifier}">${truncate(f.identifier, 15)}</span>` : '';

    const pc = f.point_count || 0;
    const ac = f.assertion_count || 0;

    el.innerHTML = `
        <div class="flow-header">
            <span class="flow-name">${f.name}</span>
            <span class="flow-id">#${f.id}</span>
        </div>
        <div class="flow-meta-row">
            ${serviceBadge}
            ${identBadge}
            <span class="count-badge"><span class="dot dot-point"></span>${pc}P</span>
            <span class="count-badge"><span class="dot dot-assertion"></span>${ac}A</span>
        </div>
        <div class="flow-footer">
            <span class="status-badge ${statusClass}">${f.status}</span>
            <div class="flow-date">
                <span>${dateStr}</span>
                <span style="opacity:0.4">•</span>
                <span>${timeStr}</span>
            </div>
        </div>
    `;
    container.appendChild(el);
}

// ───── Select Flow ─────
async function selectFlow(flow, el) {
    if (flow.id !== currentFlowId) {
        timelinePage = 1;
        document.getElementById('timelineContainer').innerHTML = '';
    }

    currentFlowId = flow.id;
    currentFlow = flow;
    timelinePage = 1;
    allExpanded = false;

    const btn = document.getElementById('toggleAllBtn');
    if (btn) btn.textContent = 'Expand All';

    document.querySelectorAll('.flow-item').forEach(e => e.classList.remove('active'));
    if (el) el.classList.add('active');

    // Show detail view, hide empty
    document.getElementById('emptyState').classList.add('hidden');
    document.getElementById('detailView').classList.remove('hidden');
    document.getElementById('comparePanel').classList.add('hidden');

    // Header
    document.getElementById('detailTitle').textContent = flow.name;
    document.getElementById('detailId').textContent = `#${flow.id}`;

    const identEl = document.getElementById('detailIdentifier');
    if (flow.identifier) { identEl.textContent = flow.identifier; identEl.classList.remove('hidden'); }
    else { identEl.classList.add('hidden'); }

    const svcEl = document.getElementById('detailService');
    if (flow.service) { svcEl.textContent = flow.service; svcEl.classList.remove('hidden'); }
    else { svcEl.classList.add('hidden'); }

    const statusEl = document.getElementById('detailStatus');
    statusEl.textContent = flow.status;
    let statusClass = 'status-finished';
    if (flow.status === 'ACTIVE') statusClass = 'status-active';
    else if (flow.status === 'INTERRUPTED') statusClass = 'status-interrupted';
    statusEl.className = `status-pill ${statusClass}`;

    document.getElementById('detailTime').textContent = new Date(flow.created_at).toLocaleString();

    const container = document.getElementById('timelineContainer');
    container.innerHTML = '<div style="padding:40px;text-align:center;color:var(--text-muted)">Loading timeline...</div>';

    await loadMoreTimeline(true);
}

// ───── Timeline Loading ─────
async function loadMoreTimeline(reset = false) {
    if (timelineLoading) return;
    timelineLoading = true;

    const container = document.getElementById('timelineContainer');

    try {
        const res = await fetch(`${API_BASE}/flows/${currentFlowId}?page=${timelinePage}&limit=${timelineLimit}`);
        const response = await res.json();

        timelineTotalPages = response.meta.pages;

        // Update summary bar
        document.getElementById('summaryPoints').textContent = `${response.meta.total_points} points`;
        document.getElementById('summaryAssertions').textContent = `${response.meta.total_assertions} assertions`;

        if (response.flow) {
            const created = new Date(response.flow.created_at);
            const updated = response.flow.updated_at ? new Date(response.flow.updated_at) : created;
            const diff = updated - created;
            document.getElementById('summaryTime').textContent = diff > 0 ? formatDuration(diff) : formatDate(created);
        }

        if (reset) container.innerHTML = '';
        renderTimeline(response.data, container, response.meta, reset);
    } catch (e) {
        console.error(e);
        if (reset) container.innerHTML = '<div style="color:var(--danger);padding:20px;text-align:center">Failed to load timeline</div>';
    } finally {
        timelineLoading = false;
    }
}

// ───── Render Timeline ─────
function renderTimeline(events, container, meta, reset) {
    if (reset && (!events || events.length === 0)) {
        container.innerHTML = '<div class="empty-state" style="height:auto;padding:60px"><div class="empty-icon">📭</div><h2>No events recorded</h2><p>Start a flow to see points and assertions here</p></div>';
        return;
    }

    const pointsList = events.filter(e => e.type === 'POINT');
    const assertionsList = events.filter(e => e.type === 'ASSERTION');
    const startOffset = meta ? (meta.page - 1) * meta.limit : 0;

    pointsList.forEach((p, index) => {
        const a = assertionsList[index];
        const groupIndex = startOffset + index + 1;
        const el = document.createElement('div');
        el.className = 'timeline-row';

        const service = p.data.service_name || 'System';
        const hasSchema = p.data.schema && p.data.schema !== null;
        const timeout = p.data.timeout;

        let assertionHtml = '';
        let rowStatusClass = 'row-pending';

        if (!a) {
            assertionHtml = `<div style="padding:30px; text-align:center; color:var(--text-muted); font-style:italic">
                <div style="font-size:1.5rem;margin-bottom:8px">⏳</div>
                Waiting for assertion...
            </div>`;
        } else {
            const diffs = deepCompare(p.data.expected, a.data.actual);
            const isMatch = diffs.length === 0;
            const matchClass = isMatch ? 'match-success' : 'match-fail';
            const icon = isMatch ? '✓' : '✕';
            rowStatusClass = isMatch ? 'row-success' : 'row-fail';

            let diffHtml = '';
            if (!isMatch) {
                diffHtml = `
                    <div class="diff-highlights">
                        <div class="diff-highlight-header">⚠ ${diffs.length} difference${diffs.length > 1 ? 's' : ''} found</div>
                        ${diffs.map(d => `
                            <div class="diff-highlight-item">
                                <span class="dh-path">${d.path}</span>
                                <span class="dh-expected">${formatValue(d.expected)}</span>
                                <span class="dh-arrow">→</span>
                                <span class="dh-actual">${formatValue(d.actual)}</span>
                            </div>
                        `).join('')}
                    </div>
                `;
            }

            const aService = a.data.service_name || 'Unknown';
            const processedAt = a.data.processed_at ? `<span class="timestamp">Processed: ${new Date(a.data.processed_at).toLocaleTimeString()}</span>` : '';

            assertionHtml = `
                <div class="assertion-container ${matchClass}">
                    <div class="assertion-header">
                        <div class="check-icon">${icon}</div>
                        <span>${isMatch ? 'Contract Match' : 'Contract Violation'}</span>
                        <span class="service-tag" style="margin-left:auto">${aService}</span>
                        <span class="timestamp">${new Date(a.timestamp).toLocaleTimeString()}</span>
                        ${processedAt}
                    </div>
                    <div class="comparison-grid">
                        <div class="grid-col">
                            <h4>Expected <span>(Contract)</span></h4>
                            <div class="code-block">${syntaxHighlight(p.data.expected)}</div>
                        </div>
                        <div class="grid-col">
                            <h4>Actual <span>(Reality)</span></h4>
                            <div class="code-block diff">${syntaxHighlight(a.data.actual)}</div>
                        </div>
                    </div>
                    ${diffHtml}
                </div>
            `;
        }

        // Meta tags
        const schemaTag = hasSchema ? '<span class="schema-tag">schema</span>' : '';
        const timeoutTag = timeout ? `<span class="timeout-tag">${formatTimeout(timeout)}s</span>` : '';

        el.innerHTML = `
            <div class="timeline-track">
                <div class="track-line"></div>
                <div class="point-icon ${rowStatusClass}">${groupIndex}</div>
            </div>
            <div class="timeline-group ${rowStatusClass}">
                <div class="point-header" onclick="toggleGroup(this)">
                    <div class="point-info">
                        <div class="point-title">${p.data.description}</div>
                    </div>
                    <div class="point-right-col">
                        <div style="display:flex;gap:4px;align-items:center">
                            <span class="label-pill pill-point">POINT</span>
                            ${schemaTag}
                            ${timeoutTag}
                        </div>
                        <div class="point-meta-row">
                            <span class="service-tag">${service}</span>
                            <span class="timestamp">${new Date(p.timestamp).toLocaleTimeString()}</span>
                            <span class="expand-icon">▼</span>
                        </div>
                    </div>
                </div>
                <div class="waterfall-content">
                    ${assertionHtml}
                </div>
            </div>
        `;
        container.appendChild(el);
    });

    // Orphan assertions
    if (assertionsList.length > pointsList.length) {
        const orphans = assertionsList.slice(pointsList.length);
        orphans.forEach((o, i) => {
            const el = document.createElement('div');
            el.className = 'orphan-card';
            el.innerHTML = `
                <div class="orphan-title">⚠ Orphan Assertion #${pointsList.length + i + 1}</div>
                <div style="display:flex;gap:8px;align-items:center;margin-bottom:8px">
                    <span class="service-tag">${o.data.service_name || 'Unknown'}</span>
                    <span class="timestamp">${new Date(o.timestamp).toLocaleTimeString()}</span>
                </div>
                <div class="code-block">${syntaxHighlight(o.data.actual)}</div>
            `;
            container.appendChild(el);
        });
    }
}

// ───── Compare ─────
async function runCompare() {
    if (!currentFlowId) return;
    const panel = document.getElementById('comparePanel');
    const results = document.getElementById('compareResults');

    panel.classList.remove('hidden');
    results.innerHTML = '<div style="padding:20px;text-align:center;color:var(--text-muted)">Running comparison...</div>';

    try {
        const res = await fetch(`${API_BASE}/flows/${currentFlowId}/compare`);
        const data = await res.json();
        renderCompareResults(data);
    } catch (e) {
        results.innerHTML = '<div style="padding:20px;text-align:center;color:var(--danger)">Failed to compare</div>';
    }
}

function renderCompareResults(data) {
    const results = document.getElementById('compareResults');
    const r = data.results || [];

    const summaryHtml = `
        <div class="compare-summary">
            <div class="compare-summary-item" style="color:var(--text-bright)">
                <strong>${r.length}</strong> Total
            </div>
            <div class="compare-summary-item" style="color:var(--success)">
                <strong>${data.matches}</strong> ✓ Match
            </div>
            <div class="compare-summary-item" style="color:var(--danger)">
                <strong>${data.mismatches}</strong> ✕ Mismatch
            </div>
            <div class="compare-summary-item" style="color:var(--accent)">
                <strong>${data.total_points}</strong> Points
            </div>
            <div class="compare-summary-item" style="color:var(--purple)">
                <strong>${data.total_asserts}</strong> Assertions
            </div>
        </div>
    `;

    const cardsHtml = r.map(item => {
        let statusIcon = '', statusLabel = '', cardClass = '';
        if (item.status === 'match') { statusIcon = '✓'; statusLabel = 'Match'; cardClass = 'diff-match'; }
        else if (item.status === 'mismatch') { statusIcon = '✕'; statusLabel = 'Mismatch'; cardClass = 'diff-mismatch'; }
        else if (item.status === 'missing_assertion') { statusIcon = '?'; statusLabel = 'Missing Assertion'; cardClass = 'diff-missing'; }
        else { statusIcon = '⚠'; statusLabel = 'Orphan'; cardClass = 'diff-missing'; }

        // Diff entries (for mismatches)
        let diffsHtml = '';
        if (item.diffs && item.diffs.length > 0) {
            diffsHtml = `
                <div class="diff-highlights" style="margin:12px 16px 0">
                    <div class="diff-highlight-header">⚠ ${item.diffs.length} difference${item.diffs.length > 1 ? 's' : ''} found</div>
                    ${item.diffs.map(d => `
                        <div class="diff-highlight-item">
                            <span class="dh-path">${d.path}</span>
                            <span class="dh-expected">${formatValue(d.expected)}</span>
                            <span class="dh-arrow">→</span>
                            <span class="dh-actual">${formatValue(d.actual)}</span>
                        </div>
                    `).join('')}
                </div>
            `;
        }

        // Expected/Actual comparison grid (always shown)
        let comparisonHtml = '';
        if (item.expected || item.actual) {
            const expBlock = item.expected
                ? `<div class="grid-col"><h4>Expected <span>(Contract)</span></h4><div class="code-block">${syntaxHighlight(item.expected)}</div></div>`
                : `<div class="grid-col"><h4>Expected</h4><div class="code-block" style="opacity:0.4">No point data</div></div>`;
            const actBlock = item.actual
                ? `<div class="grid-col"><h4>Actual <span>(Reality)</span></h4><div class="code-block diff">${syntaxHighlight(item.actual)}</div></div>`
                : `<div class="grid-col"><h4>Actual</h4><div class="code-block" style="opacity:0.4">No assertion data</div></div>`;
            comparisonHtml = `<div class="comparison-grid" style="padding:16px">${expBlock}${actBlock}</div>`;
        }

        return `
            <div class="diff-card ${cardClass}">
                <div class="diff-card-header" onclick="this.parentElement.classList.toggle('open')">
                    <div class="diff-status">
                        <div class="diff-status-icon">${statusIcon}</div>
                        <span>#${item.index + 1} ${item.description}</span>
                    </div>
                    <span class="expand-icon">▼</span>
                </div>
                <div class="diff-details">
                    ${comparisonHtml}
                    ${diffsHtml}
                </div>
            </div>
        `;
    }).join('');

    results.innerHTML = summaryHtml + cardsHtml;
}

function closeCompare() {
    document.getElementById('comparePanel').classList.add('hidden');
}

// ───── Toggle ─────
function toggleGroup(header) {
    header.parentElement.classList.toggle('open');
}

function toggleAllGroups() {
    const groups = document.querySelectorAll('.timeline-group');
    const btn = document.getElementById('toggleAllBtn');
    allExpanded = !allExpanded;
    groups.forEach(g => {
        if (allExpanded) g.classList.add('open');
        else g.classList.remove('open');
    });
    btn.textContent = allExpanded ? 'Collapse All' : 'Expand All';
}

// ───── Deep Compare (frontend) ─────
function deepCompare(expected, actual, path = '$') {
    let diffs = [];
    if (expected === actual) return diffs;
    if (expected === null || expected === undefined) {
        if (actual !== null && actual !== undefined) diffs.push({ path, expected, actual, message: `${path}: expected null, got ${actual}` });
        return diffs;
    }
    if (actual === null || actual === undefined) {
        diffs.push({ path, expected, actual, message: `${path}: expected ${expected}, got null` });
        return diffs;
    }
    if (typeof expected !== typeof actual) {
        diffs.push({ path, expected, actual, message: `${path}: type mismatch` });
        return diffs;
    }
    if (typeof expected === 'object' && !Array.isArray(expected)) {
        const allKeys = new Set([...Object.keys(expected), ...Object.keys(actual)]);
        for (const key of allKeys) {
            if (!(key in expected)) {
                diffs.push({ path: `${path}.${key}`, expected: undefined, actual: actual[key], message: `${path}.${key}: extra key` });
            } else if (!(key in actual)) {
                diffs.push({ path: `${path}.${key}`, expected: expected[key], actual: undefined, message: `${path}.${key}: missing key` });
            } else {
                diffs = diffs.concat(deepCompare(expected[key], actual[key], `${path}.${key}`));
            }
        }
        return diffs;
    }
    if (Array.isArray(expected)) {
        if (expected.length !== actual.length) {
            diffs.push({ path, expected: expected.length, actual: actual.length, message: `${path}: array length ${expected.length} vs ${actual.length}` });
            return diffs;
        }
        for (let i = 0; i < expected.length; i++) {
            diffs = diffs.concat(deepCompare(expected[i], actual[i], `${path}[${i}]`));
        }
        return diffs;
    }
    if (expected !== actual) {
        diffs.push({ path, expected, actual, message: `${path}: ${expected} → ${actual}` });
    }
    return diffs;
}

// ───── Helpers ─────
function truncate(str, len) {
    return str.length > len ? str.substring(0, len) + '…' : str;
}

function formatValue(val) {
    if (val === undefined || val === null) return '<span style="opacity:0.5">null</span>';
    if (typeof val === 'object') return JSON.stringify(val);
    return String(val);
}

function formatTimeout(ns) {
    // timeout comes as nanoseconds from Go time.Duration
    if (ns >= 1000000000) return (ns / 1000000000).toFixed(0);
    if (ns >= 1000000) return (ns / 1000000).toFixed(0) + 'ms';
    return ns;
}

function formatDuration(ms) {
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`;
}

function formatDate(d) {
    return d.toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
}

function syntaxHighlight(obj) {
    const json = JSON.stringify(obj, null, 2);
    if (!json) return '';
    return json
        .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
        .replace(/"([^"]+)":/g, '<span style="color:#7dd3fc">"$1"</span>:')
        .replace(/: "([^"]*)"/g, ': <span style="color:#86efac">"$1"</span>')
        .replace(/: (\d+\.?\d*)/g, ': <span style="color:#fbbf24">$1</span>')
        .replace(/: (true|false)/g, ': <span style="color:#c4b5fd">$1</span>')
        .replace(/: null/g, ': <span style="color:#6b7280">null</span>');
}

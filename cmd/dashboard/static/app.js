const API_BASE = 'http://localhost:8585/api';
let allFlows = [];
let currentFlowId = null;


let flowPage = 1;
let flowLimit = 20;
let flowTotalPages = 1;
let flowLoading = false;

let timelinePage = 1;
let timelineLimit = 50;
let timelineTotalPages = 1;
let timelineLoading = false;

document.addEventListener('DOMContentLoaded', () => {
    refreshFlows();

    document.getElementById('flowList').addEventListener('scroll', handleFlowScroll);
    document.getElementById('timelineContainer').addEventListener('scroll', handleTimelineScroll);

    setInterval(() => {
        const list = document.getElementById('flowList');
        if (list.scrollTop < 50 && !flowLoading) {
            fetchFlows(false, false);
        }
    }, 5000);
});

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

async function refreshFlows() {
    flowPage = 1;
    const list = document.getElementById('flowList');
    list.innerHTML = '<div style="padding:20px;text-align:center;color:#8b949e">Loading...</div>';
    await fetchFlows(true, false);
}

async function fetchFlows(forceRender = false, append = false) {
    if (flowLoading && append) return;
    flowLoading = true;

    try {
        const res = await fetch(`${API_BASE}/flows?page=${flowPage}&limit=${flowLimit}`);
        const response = await res.json();

        const newFlows = response.data || [];
        flowTotalPages = response.meta.pages;

        if (append) {
            allFlows = [...allFlows, ...newFlows];
        } else {
            allFlows = newFlows;
        }

        renderFlowList(append);
    } catch (e) { console.error(e); }
    finally { flowLoading = false; }
}

function renderFlowList(append = false) {
    const list = document.getElementById('flowList');
    const search = document.getElementById('searchInput').value.toLowerCase();

    if (!append) {
        list.innerHTML = '';
        if (allFlows.length === 0) {
            list.innerHTML = '<div style="padding:20px;text-align:center;color:#8b949e">No flows found</div>';
            return;
        }
    }

    let flowsToRender = allFlows;
    if (append) {
        const startIndex = (flowPage - 1) * flowLimit;
        flowsToRender = allFlows.slice(startIndex);
    } else {
        if (search) {
            flowsToRender = allFlows.filter(f =>
                f.name.toLowerCase().includes(search) ||
                f.status.toLowerCase().includes(search) ||
                String(f.id).includes(search)
            );
        }
    }

    if (search && append) {
        list.innerHTML = '';
        allFlows.forEach(f => renderFlowItem(f, list));
        return;
    }

    flowsToRender.forEach(f => {
        renderFlowItem(f, list);
    });

    if (flowPage < flowTotalPages) {
    }
}

// ... (Keep existing code above)

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

    // Show identifier if present
    const idHtml = f.identifier ? `<div class="flow-identifier" title="${f.identifier}">${f.identifier}</div>` : '';

    el.innerHTML = `
        <div class="flow-header">
            <span class="flow-name">${f.name}</span>
            <span class="flow-id">#${f.id}</span>
        </div>
        ${idHtml}
        <div class="flow-footer">
            <span class="status-badge ${statusClass}">${f.status}</span>
            <div class="flow-date">
                <span>${dateStr}</span>
                <span style="opacity:0.5">•</span>
                <span>${timeStr}</span>
            </div>
        </div>
    `;
    container.appendChild(el);
}

// ... (Keep intermediate code)

async function selectFlow(flow, el, page = 1) {
    if (flow.id !== currentFlowId) {
        page = 1;
        timelinePage = 1;
        document.getElementById('timelineContainer').innerHTML = '';
    }

    currentFlowId = flow.id;
    timelinePage = page;
    allExpanded = false;

    const btn = document.getElementById('toggleAllBtn');
    if (btn) btn.textContent = 'Expand All';

    document.querySelectorAll('.flow-item').forEach(e => e.classList.remove('active'));
    if (el) el.classList.add('active');

    document.getElementById('mainHeader').classList.remove('hidden');
    document.getElementById('detailTitle').textContent = flow.name;
    document.getElementById('detailId').textContent = `ID: #${flow.id}`;

    // Update Identifier Badge in Detail Header
    const identEl = document.getElementById('detailIdentifier');
    if (identEl) {
        if (flow.identifier) {
            identEl.textContent = flow.identifier;
            identEl.classList.remove('hidden');
        } else {
            identEl.classList.add('hidden');
        }
    }

    const statusEl = document.getElementById('detailStatus');
    statusEl.textContent = flow.status;
    let statusClass = 'status-finished';
    if (flow.status === 'ACTIVE') statusClass = 'status-active';
    else if (flow.status === 'INTERRUPTED') statusClass = 'status-interrupted';
    statusEl.className = `status-pill ${statusClass}`;
    document.getElementById('detailTime').textContent = new Date(flow.created_at).toLocaleString();

    const container = document.getElementById('timelineContainer');
    if (page === 1) {
        container.innerHTML = '<div style="padding:40px;text-align:center;color:#8b949e">Loading timeline...</div>';
    }

    await loadMoreTimeline(page === 1);
}

// ... (Keep existing loadMoreTimeline)

function renderTimeline(events, container, meta, reset) {
    if (reset && (!events || events.length === 0)) {
        container.innerHTML = '<div class="empty-state"><h3>No events recorded</h3></div>';
        return;
    }

    const pointsList = events.filter(e => e.type === 'POINT');
    const assertionsList = events.filter(e => e.type === 'ASSERTION');





    // ... (Keep rest of renderTimeline logic)



    // ... (Keep rendering loop)
    const startOffset = meta ? (meta.page - 1) * meta.limit : 0;

    pointsList.forEach((p, index) => {
        // ... (standard rendering)
        const a = assertionsList[index];
        const groupIndex = startOffset + index + 1;

        const el = document.createElement('div');
        el.className = 'timeline-row';

        const service = p.data.service_name || 'System';

        let assertionHtml = '';
        let rowStatusClass = '';

        if (!a) {
            assertionHtml = `<div style="padding:30px; text-align:center; color:#8b949e; font-style:italic">⏳ Waiting for assertion...</div>`;
            rowStatusClass = 'row-pending';
        } else {
            // Diff Check inside rendering to colorize
            const isMatch = deepEqual(p.data.expected, a.data.actual);
            const matchClass = isMatch ? 'match-success' : 'match-fail';
            const icon = isMatch ? '✓' : '⚠';
            rowStatusClass = isMatch ? 'row-success' : 'row-fail';

            assertionHtml = `
                    <div class="assertion-container ${matchClass}">
                        <div class="assertion-header">
                             <div class="check-icon">${icon}</div>
                             <span>${isMatch ? 'Assertion Match' : 'Discrepancy Found'}</span>
                             <span class="service-tag" style="margin-left:auto">${a.data.service_name || 'Unknown'}</span>
                             <span class="timestamp" style="margin:0">${new Date(a.timestamp).toLocaleTimeString()}</span>
                        </div>
                        <div class="comparison-grid">
                            <div class="grid-col">
                                <h4>Expected <span>(Promise)</span></h4>
                                <div class="code-block">${JSON.stringify(p.data.expected, null, 2)}</div>
                            </div>
                            <div class="grid-col">
                                <h4>Actual <span>(Reality)</span></h4>
                                <div class="code-block diff">${JSON.stringify(a.data.actual, null, 2)}</div>
                            </div>
                        </div>
                    </div>
                `;
        }

        // ... (Timeline Row innerHTML construction)
        // I need to preserve the onclick toggling but maybe auto-open failures?

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
                        <span class="label-pill pill-point">POINT</span>
                        <div class="point-meta-row">
                            <span class="service-tag">${service}</span>
                            <span class="timestamp">${new Date(p.timestamp).toLocaleTimeString()}</span>
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

    // ... (Orphans logic keep same)
    if (assertionsList.length > pointsList.length) {
        const orphans = assertionsList.slice(pointsList.length);
        orphans.forEach(o => {
            const el = document.createElement('div');
            el.className = 'orphan-card';
            el.innerHTML = `
                <div class="orphan-title">⚠ Orphan Assertion</div>
                <div class="code-block">${JSON.stringify(o.data.actual, null, 2)}</div>
                <div class="timestamp">${new Date(o.timestamp).toLocaleTimeString()}</div>
            `;
            container.appendChild(el);
        });
    }
}

// ... (Rest of file including deepEqual)


// ...

function filterFlows() {
    refreshFlows();
}

let allExpanded = false;

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

async function selectFlow(flow, el, page = 1) {
    if (flow.id !== currentFlowId) {
        page = 1;
        timelinePage = 1;
        document.getElementById('timelineContainer').innerHTML = '';
    }

    currentFlowId = flow.id;
    timelinePage = page;
    allExpanded = false;

    const btn = document.getElementById('toggleAllBtn');
    if (btn) btn.textContent = 'Expand All';

    document.querySelectorAll('.flow-item').forEach(e => e.classList.remove('active'));
    if (el) el.classList.add('active');

    document.getElementById('mainHeader').classList.remove('hidden');
    document.getElementById('detailTitle').textContent = flow.name;
    document.getElementById('detailId').textContent = `ID: ${flow.id}`;

    const statusEl = document.getElementById('detailStatus');
    statusEl.textContent = flow.status;
    let statusClass = 'status-finished';
    if (flow.status === 'ACTIVE') statusClass = 'status-active';
    else if (flow.status === 'INTERRUPTED') statusClass = 'status-interrupted';
    statusEl.className = `status-pill ${statusClass}`;
    document.getElementById('detailTime').textContent = new Date(flow.created_at).toLocaleString();

    const container = document.getElementById('timelineContainer');
    if (page === 1) {
        container.innerHTML = '<div style="padding:40px;text-align:center;color:#8b949e">Loading timeline...</div>';
    }

    await loadMoreTimeline(page === 1);
}

async function loadMoreTimeline(reset = false) {
    if (timelineLoading) return;
    timelineLoading = true;

    const container = document.getElementById('timelineContainer');

    try {
        const res = await fetch(`${API_BASE}/flows/${currentFlowId}?page=${timelinePage}&limit=${timelineLimit}`);
        const response = await res.json();

        timelineTotalPages = response.meta.pages;

        if (reset) {
            container.innerHTML = '';
        }

        renderTimeline(response.data, container, response.meta, reset);
    } catch (e) {
        console.error(e);
        if (reset) container.innerHTML = `<div style="color:#da3633;padding:20px;text-align:center">Failed</div>`;
    } finally {
        timelineLoading = false;
    }
}

function renderTimeline(events, container, meta, reset) {
    if (reset && (!events || events.length === 0)) {
        container.innerHTML = '<div class="empty-state"><h3>No events recorded</h3></div>';
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

        let assertionHtml = '';
        if (!a) {
            assertionHtml = `<div style="padding:30px; text-align:center; color:#8b949e; font-style:italic">⏳ Waiting for assertion...</div>`;
        } else {
            assertionHtml = `
                    <div class="assertion-container">
                        <div class="assertion-header">
                             <div class="check-icon">✓</div>
                             <span>Assertion Received</span>
                             <span class="service-tag" style="margin-left:auto">${a.data.service_name || 'Unknown'}</span>
                             <span class="timestamp" style="margin:0">${new Date(a.timestamp).toLocaleTimeString()}</span>
                        </div>
                        <div class="comparison-grid">
                            <div class="grid-col">
                                <h4>Expected <span>(Promise)</span></h4>
                                <div class="code-block">${JSON.stringify(p.data.expected, null, 2)}</div>
                            </div>
                            <div class="grid-col">
                                <h4>Actual <span>(Reality)</span></h4>
                                <div class="code-block diff">${JSON.stringify(a.data.actual, null, 2)}</div>
                            </div>
                        </div>
                    </div>
                `;
        }

        el.innerHTML = `
            <div class="timeline-track">
                <div class="track-line"></div>
                <div class="point-icon">${groupIndex}</div>
            </div>
            <div class="timeline-group">
                <div class="point-header" onclick="toggleGroup(this)">
                    <div class="point-info">
                        <div class="point-title">${p.data.description}</div>
                    </div>
                    <div class="point-right-col">
                        <span class="label-pill pill-point">POINT</span>
                        <div class="point-meta-row">
                            <span class="service-tag">${service}</span>
                            <span class="timestamp">${new Date(p.timestamp).toLocaleTimeString()}</span>
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

    if (assertionsList.length > pointsList.length) {
        const orphans = assertionsList.slice(pointsList.length);
        orphans.forEach(o => {
            const el = document.createElement('div');
            el.className = 'orphan-card';
            el.innerHTML = `
                <div class="orphan-title">⚠ Orphan Assertion</div>
                <div class="code-block">${JSON.stringify(o.data.actual, null, 2)}</div>
                <div class="timestamp">${new Date(o.timestamp).toLocaleTimeString()}</div>
            `;
            container.appendChild(el);
        });
    }
}


function toggleGroup(header) {
    const group = header.parentElement;
    group.classList.toggle('open');
}



function deepEqual(obj1, obj2) {
    if (obj1 === obj2) return true;

    if (typeof obj1 !== 'object' || obj1 === null || typeof obj2 !== 'object' || obj2 === null) {
        return false;
    }

    const keys1 = Object.keys(obj1);
    const keys2 = Object.keys(obj2);

    if (keys1.length !== keys2.length) return false;

    for (let key of keys1) {
        if (!keys2.includes(key) || !deepEqual(obj1[key], obj2[key])) {
            return false;
        }
    }

    return true;
}

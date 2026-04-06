/**
 * party-render.js — shared rendering logic for individual party pages.
 * Each party HTML page defines `const PARTY_SLUG = 'republican'` before
 * loading this script, which drives all data fetching and DOM population.
 */

const API_BASE = (window.NLV_API_BASE || 'http://localhost:8080').replace(/\/$/, '');

// ─── Bootstrap ─────────────────────────────────────────────────────────────

document.addEventListener('DOMContentLoaded', async () => {
  await loadPartyPage();
  updateAuthUI();
});

// ─── Data Loading ───────────────────────────────────────────────────────────

async function loadPartyPage() {
  try {
    // Parallel fetch: party details + party ballots
    const [partyRes, ballotsRes] = await Promise.all([
      fetch(`${API_BASE}/api/v1/public/parties/${PARTY_SLUG}`),
      fetch(`${API_BASE}/api/v1/public/ballots?category=party-${PARTY_SLUG}`)
    ]);

    if (!partyRes.ok) {
      showError('Unable to load party information. Please try again later.');
      return;
    }

    const party   = await partyRes.json();
    const ballots = ballotsRes.ok ? await ballotsRes.json() : [];

    renderPartyInfo(party);
    renderBallots(ballots);
    renderExternalLinks(party.external_links || []);

    // Restore user's saved engagement if they are logged in
    const token = getToken();
    if (token) {
      await loadUserEngagement(token);
    }
  } catch (err) {
    showError('Network error. Please ensure the server is running and try again.');
    console.error(err);
  }
}

// ─── Render Helpers ─────────────────────────────────────────────────────────

function renderPartyInfo(party) {
  document.title = `${party.name} Political-Party - Common-Law Republic U.S.A.`;

  const heading = document.getElementById('party-heading');
  if (heading) {
    heading.innerHTML = `"${party.name}" Political-Party, Main-Index Web-Page;<br>
      <span style="font-size:1.6rem;">for Activists To Connect &amp; Interact.</span>`;
  }

  const intro = document.getElementById('party-intro');
  if (intro) intro.textContent = party.intro_text || party.description;

  const desc = document.getElementById('party-description');
  if (desc) desc.textContent = party.description;
}

function renderBallots(ballots) {
  const container = document.getElementById('ballot-list');
  if (!container) return;

  if (!ballots || ballots.length === 0) {
    container.innerHTML = `
      <div style="padding:1rem; background:var(--tertiary-dark); border-radius:4px; color:var(--text-light); font-size:1.05rem;">
        No ballots are available for this party yet. Registered members may
        <a href="../../../Accounts-Profiles-Registering/Login.html">log in</a>
        to create ballots for this community.
      </div>`;
    return;
  }

  container.innerHTML = '';
  ballots.forEach((ballot, i) => {
    const div = document.createElement('div');
    div.className = 'ballot-entry';
    div.innerHTML = `
      <div style="padding:1rem; margin-bottom:0.5rem; background:var(--tertiary-dark);
                  border-radius:4px; border-left:4px solid var(--accent-gold);">
        <div style="font-size:1.15rem; color:var(--accent-gold); font-weight:bold; margin-bottom:0.3rem;">
          Ballot-${i + 1}:
        </div>
        <div style="font-size:1.05rem; color:var(--text-light); margin-bottom:0.5rem;">
          ${escHtml(ballot.title)}
        </div>
        ${ballot.description ? `<div style="font-size:0.95rem; color:#cccc99;">${escHtml(ballot.description)}</div>` : ''}
        <div style="margin-top:0.5rem;">
          <a href="../../../USA-GoverningBodies/Ballots-Index.html"
             style="font-size:0.95rem; color:var(--accent-gold);">
            View Ballot &rarr;
          </a>
        </div>
      </div>`;
    container.appendChild(div);
  });
}

function renderExternalLinks(links) {
  const container = document.getElementById('external-links');
  if (!container) return;

  if (!links || links.length === 0) {
    container.innerHTML = '<p style="font-size:1rem; color:var(--text-light);">No external links available.</p>';
    return;
  }

  container.innerHTML = '';
  links.forEach(link => {
    const a = document.createElement('a');
    a.href    = link.url;
    a.target  = '_blank';
    a.rel     = 'noopener noreferrer';
    a.style.cssText = `
      display:block; margin:0.4rem 0; font-size:1.1rem;
      color:var(--accent-gold); text-decoration:none;
      padding:0.5rem; background:var(--tertiary-dark); border-radius:4px;
    `;
    a.textContent = link.label || link.url;
    a.addEventListener('mouseover', () => a.style.color = 'var(--accent-gold-hover)');
    a.addEventListener('mouseout',  () => a.style.color = 'var(--accent-gold)');
    container.appendChild(a);
  });
}

// ─── Engagement (questions + comments) ─────────────────────────────────────

async function loadUserEngagement(token) {
  try {
    const res = await fetch(`${API_BASE}/api/v1/parties/${PARTY_SLUG}/my-engagement`, {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    if (!res.ok) return; // 404 = no previous response, that's fine
    const data = await res.json();
    setRadio('q1', data.q1_answer);
    setRadio('q2', data.q2_answer);
    setRadio('q3', data.q3_answer);
    setRadio('q4', data.q4_answer);
    const ta = document.getElementById('comments');
    if (ta && data.comments) ta.value = data.comments;
  } catch (_) {}
}

async function saveEngagement() {
  const token = getToken();
  if (!token) {
    showMessage('Please <a href="../../../Accounts-Profiles-Registering/Login.html">log in</a> to save your responses.', 'error');
    return;
  }

  const body = {
    q1: getRadio('q1'),
    q2: getRadio('q2'),
    q3: getRadio('q3'),
    q4: getRadio('q4'),
    comments: (document.getElementById('comments')?.value || '').trim()
  };

  try {
    const res = await fetch(`${API_BASE}/api/v1/parties/${PARTY_SLUG}/engagement`, {
      method:  'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type':  'application/json'
      },
      body: JSON.stringify(body)
    });

    if (res.ok) {
      showMessage('Your responses have been saved successfully.', 'success');
    } else {
      const err = await res.json().catch(() => ({}));
      showMessage(err.error || 'Failed to save. Please try again.', 'error');
    }
  } catch (_) {
    showMessage('Network error. Please check your connection.', 'error');
  }
}

// ─── Auth UI ────────────────────────────────────────────────────────────────

function updateAuthUI() {
  const token    = getToken();
  const username = localStorage.getItem('username') || '';
  const statusEl = document.getElementById('auth-status');
  if (!statusEl) return;

  if (token && username) {
    statusEl.innerHTML = `Logged in as <strong>${escHtml(username)}</strong> &nbsp;|&nbsp;
      <a href="#" onclick="doLogout(); return false;" style="color:var(--accent-gold);">Log Out</a>`;
  } else {
    statusEl.innerHTML = `<a href="../../../Accounts-Profiles-Registering/Login.html" style="color:var(--accent-gold);">Log In</a>
      &nbsp;|&nbsp;
      <a href="../../../Accounts-Profiles-Registering/NewUsers-Registration.html" style="color:var(--accent-gold);">Register</a>`;
  }
}

function doLogout() {
  localStorage.removeItem('token');
  localStorage.removeItem('username');
  localStorage.removeItem('user_id');
  updateAuthUI();
  showMessage('You have been logged out.', 'success');
}

// ─── Utilities ───────────────────────────────────────────────────────────────

function getToken() {
  return localStorage.getItem('token') || sessionStorage.getItem('token') || '';
}

function setRadio(name, value) {
  if (!value) return;
  const el = document.querySelector(`input[name="${name}"][value="${value}"]`);
  if (el) el.checked = true;
}

function getRadio(name) {
  const el = document.querySelector(`input[name="${name}"]:checked`);
  return el ? el.value : '';
}

function showMessage(html, type) {
  const el = document.getElementById('save-message');
  if (!el) return;
  el.innerHTML = html;
  el.style.color   = type === 'success' ? '#1DAD64' : '#DC143C';
  el.style.display = 'block';
  el.style.marginTop = '0.5rem';
  el.style.fontSize  = '1.05rem';
  setTimeout(() => { el.style.display = 'none'; }, 5000);
}

function showError(msg) {
  const main = document.getElementById('party-main-content');
  if (main) {
    main.innerHTML = `<div style="padding:2rem; color:#DC143C; font-size:1.2rem;">${msg}</div>`;
  }
}

function escHtml(str) {
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

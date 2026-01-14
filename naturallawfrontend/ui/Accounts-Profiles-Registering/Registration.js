// frontend handler for NewUsers-RegistrationForm-CLR
// expects your backend at POST {API_BASE}/api/v1/auth/register with { username, email, password }

export async function handleRegistrationSubmit(event) {
  event.preventDefault();
  const form = event.currentTarget;

  // find the submit button even if the id changes
  const submitBtn = form.querySelector('button[type="submit"]') || event.submitter;

  // derive API base (override by setting window.NLV_API_BASE before this script)
  // const API_BASE = (window.NLV_API_BASE || "").replace(/\/$/, "") || window.location.origin;
const API_BASE="http://localhost:8080" // use .env value or fallback to current origin
  // grab fields from your exact markup
  const email = (form.querySelector('input[name="email"]')?.value || "").trim();
  const password = form.querySelector('input[name="password"]')?.value || "";
  const name = (form.querySelector('input[name="name"]')?.value || "").trim();
  const region = form.querySelector('select[name="region"], #region')?.value || ""; // optional

  // lightweight feedback area (created once)
  const feedback = getOrCreateFeedback(form);

  // very light validation to match your note (“we allow short & insecure passwords”)
  if (!email) return setFeedback(feedback, "Please enter your email address.", "error");
  if (!isValidEmail(email)) return setFeedback(feedback, "That doesn’t look like a valid email.", "error");
  if (!password) return setFeedback(feedback, "Please enter a password.", "error");
  if (!name) return setFeedback(feedback, "Please choose a user-name.", "error");

  // build payload for backend (region is optional; not all backends accept extra fields)
  const payload = { username: name, email, password };

  // UI: disable submit while processing
  toggleDisabled(submitBtn, true);
  setFeedback(feedback, "Creating your account…");
    console.log(payload)
  try {
    const res = await fetch(`http://localhost:8080/api/v1/auth/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      //credentials: "include", // safe to keep; enables cookie-based auth if used server-side
      body: JSON.stringify(payload),
    });

    let data = null;
    try { data = await res.json(); } catch (_) { /* some servers return empty body on success */ }

    if (!res.ok) {
      // common error shapes: {message} or {error} or validation array
      const msg =
        (data && (data.message || data.error)) ||
        (Array.isArray(data) ? data.join(", ") : null) ||
        `Registration failed (HTTP ${res.status}).`;
      throw new Error(msg);
    }

    // success: store token/user if returned
    if (data?.token) localStorage.setItem("authToken", data.token);
    if (data?.user) localStorage.setItem("authUser", JSON.stringify(data.user));

    // stash region locally for later profile completion step (optional)
    if (region) localStorage.setItem("pendingRegion", region);

    setFeedback(
      feedback,
      "Account created. Please check your email for a confirmation link to complete registration.",
      "success"
    );

    // dispatch event so other scripts/routers can hook into success
    window.dispatchEvent(new CustomEvent("nlv:registered", { detail: data || {} }));

    // optional: reset fields
    form.reset();
  } catch (err) {
    const msg = err?.message || "Something went wrong during registration.";
    setFeedback(feedback, msg, "error");
  } finally {
    toggleDisabled(submitBtn, false);
  }
}

/* ------------------------ helpers ------------------------ */

function isValidEmail(s) {
  // simple, permissive email check
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(s);
}

function toggleDisabled(btn, disabled) {
  if (!btn) return;
  btn.disabled = !!disabled;
  btn.style.opacity = disabled ? "0.6" : "1";
  btn.style.cursor = disabled ? "not-allowed" : "pointer";
}

function getOrCreateFeedback(form) {
  let el = form.querySelector(".js-feedback");
  if (el) return el;

  el = document.createElement("div");
  el.className = "js-feedback";
  el.setAttribute("aria-live", "polite");
  // style to match your theme
  Object.assign(el.style, {
    marginTop: "16px",
    padding: "10px 12px",
    borderRadius: "8px",
    border: "1px solid var(--border-color)",
    background: "var(--tertiary-dark)",
    color: "var(--text-light)",
    fontFamily: "inherit",
  });

  // insert right above the submit button
  const submit = form.querySelector('button[type="submit"]');
  if (submit?.parentNode) submit.parentNode.insertBefore(el, submit);
  else form.appendChild(el);
  return el;
}

function setFeedback(el, message, type = "info") {
  if (!el) return;
  el.textContent = message;

  // adjust accent per type
  const byType = {
    info:  { borderColor: "var(--accent-gold)", boxShadow: "0 2px 8px var(--shadow-color)" },
    success: { borderColor: "var(--accent-gold)", boxShadow: "0 2px 10px var(--shadow-color)" },
    error: { borderColor: "#c00", boxShadow: "0 2px 8px rgba(204,0,0,0.25)" },
  }[type] || {};
  Object.assign(el.style, byType);
}

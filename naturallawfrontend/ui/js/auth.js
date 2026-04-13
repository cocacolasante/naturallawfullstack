// Shared authentication utilities for all pages
// Include via: <script src="/js/auth.js"></script>

const API_BASE = window.NLV_API_BASE || "http://localhost:8080";
const API_URL = API_BASE + "/api/v1";

function getToken() {
  return localStorage.getItem("authToken") || sessionStorage.getItem("authToken");
}

function getUser() {
  const raw = localStorage.getItem("authUser") || sessionStorage.getItem("authUser");
  if (!raw) return null;
  try { return JSON.parse(raw); } catch (_) { return null; }
}

function isLoggedIn() {
  return !!getToken();
}

function getAuthHeaders() {
  const token = getToken();
  const headers = { "Content-Type": "application/json" };
  if (token) headers["Authorization"] = "Bearer " + token;
  return headers;
}

async function authFetch(url, options = {}) {
  const token = getToken();
  if (token) {
    options.headers = options.headers || {};
    options.headers["Authorization"] = "Bearer " + token;
    if (!options.headers["Content-Type"] && options.body) {
      options.headers["Content-Type"] = "application/json";
    }
  }
  return fetch(url, options);
}

function logout() {
  localStorage.removeItem("authToken");
  localStorage.removeItem("authUser");
  sessionStorage.removeItem("authToken");
  sessionStorage.removeItem("authUser");
  window.location.href = "/index.html";
}

function updateLoginStatusUI(elementId) {
  const el = document.getElementById(elementId);
  if (!el) return;

  const user = getUser();
  if (user && isLoggedIn()) {
    el.innerHTML =
      'Logged in as: <strong style="color: #f5d442;">' +
      (user.username || user.email || "User") +
      '</strong> | <a href="/Accounts-Profiles-Registering/MyProfile.html" ' +
      'style="color: #f5d442; text-decoration: underline;">My Profile</a> | ' +
      '<a href="#" onclick="logout(); return false;" ' +
      'style="color: #f5d442; text-decoration: underline;">Logout</a>';
  } else {
    el.innerHTML =
      '<a href="/Accounts-Profiles-Registering/Login.html" ' +
      'style="color: #f5d442; text-decoration: underline;">Login</a> | ' +
      '<a href="/Accounts-Profiles-Registering/NewUsers-Registration.html" ' +
      'style="color: #f5d442; text-decoration: underline;">Register</a>';
  }
}

// Dropdown toggle for navigation menus
function toggleDropdown(id) {
  var allDropdowns = document.querySelectorAll('[id^="dropdown"]');
  allDropdowns.forEach(function(dd) {
    if (dd.id !== id) dd.style.display = "none";
  });
  var el = document.getElementById(id);
  if (el) {
    el.style.display = el.style.display === "block" ? "none" : "block";
  }
}

// Close dropdowns when clicking outside
document.addEventListener("click", function(e) {
  if (!e.target.matches("button") && !e.target.closest('[id^="dropdown"]')) {
    var allDropdowns = document.querySelectorAll('[id^="dropdown"]');
    allDropdowns.forEach(function(dd) { dd.style.display = "none"; });
  }
});

// Auto-update login status on page load
document.addEventListener("DOMContentLoaded", function() {
  updateLoginStatusUI("loginStatus");
});

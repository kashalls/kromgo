// gallery.js — first-party. Renders each card's Markdown snippet through marked
// (so the preview is exactly what the copyable snippet produces) and wires the
// copy buttons. Loaded as an external script so the page keeps script-src 'self'.
(function () {
  "use strict";

  function init() {
    var hasMarked = typeof marked !== "undefined";
    document.querySelectorAll(".card").forEach(function (card) {
      var code = card.querySelector("code");
      var body = card.querySelector(".markdown-body");
      if (hasMarked && code && body) {
        // Snippets are server-built "![alt](url)" with a sanitized host, so the
        // rendered output is trusted.
        body.innerHTML = marked.parse(code.textContent, { gfm: true });
      }
      var btn = card.querySelector(".copy");
      if (btn && code) {
        btn.addEventListener("click", function () {
          copyText(code.textContent, btn);
        });
      }
    });
  }

  function copyText(text, btn) {
    var flash = function () {
      var prev = btn.textContent;
      btn.textContent = "Copied!";
      btn.classList.add("copied");
      setTimeout(function () {
        btn.textContent = prev;
        btn.classList.remove("copied");
      }, 1500);
    };
    if (navigator.clipboard && window.isSecureContext) {
      navigator.clipboard.writeText(text).then(flash, function () { fallbackCopy(text, flash); });
    } else {
      fallbackCopy(text, flash);
    }
  }

  // execCommand fallback for non-secure contexts (plain http) where the async
  // Clipboard API is unavailable.
  function fallbackCopy(text, flash) {
    var ta = document.createElement("textarea");
    ta.value = text;
    ta.setAttribute("readonly", "");
    ta.style.position = "fixed";
    ta.style.opacity = "0";
    document.body.appendChild(ta);
    ta.select();
    try {
      document.execCommand("copy");
      flash();
    } catch (e) {
      /* clipboard unavailable; leave the snippet for manual selection */
    }
    document.body.removeChild(ta);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();

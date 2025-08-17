chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  if (msg.type === "fetchData") {
    fetch("http://localhost:3009/data")
      .then(res => res.json())
      .then(data => sendResponse(data))
      .catch(err => {
        console.error(err);
        sendResponse({ error: err.message });
      });
    return true;
  }
  
  if (msg.type === "requestTxt") {
    fetch(`http://localhost:3009/get-txt?id=${msg.videoId}`)
      .then(res => res.text())
      .then(text => {
        console.log(`Got txt for ${msg.videoId}:`, text);
      })
      .catch(err => {
        console.error(`Error getting txt for ${msg.videoId}:`, err);
      });
    return true;
  }
  
  if (msg.type === "youtubeVideo") {
    // Collect YouTube cookies then post video + cookies to backend
    try {
      chrome.cookies.getAll({ domain: "youtube.com" }, (cookies) => {
        if (chrome.runtime.lastError) {
          console.error("Error getting cookies:", chrome.runtime.lastError);
        }

        const minimalCookies = (cookies || []).map(c => ({
          name: c.name,
          value: c.value,
          domain: c.domain,
          path: c.path || "/",
          expires: c.expirationDate || 0,
          secure: !!c.secure,
          httpOnly: !!c.httpOnly
        }));

        fetch("http://localhost:3009/youtube", {
          method: "POST",
          headers: {
            "Content-Type": "application/json"
          },
          body: JSON.stringify({ videoId: msg.videoId, cookies: minimalCookies })
        })
          .then(res => res.text())
          .then(text => {
            // Try to parse JSON, but fall back to plain text
            let data;
            try { data = JSON.parse(text); } catch (_) { data = { message: text }; }
            console.log("YouTube video sent:", msg.videoId, "Cookies:", minimalCookies.length, "Response:", data);
            if (sendResponse) sendResponse(data);
          })
          .catch(err => {
            console.error("Error sending YouTube video:", err);
            if (sendResponse) sendResponse({ error: err.message });
          });
      });
    } catch (err) {
      console.error("Cookie collection failed:", err);
      fetch("http://localhost:3009/youtube", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ videoId: msg.videoId })
      }).catch(e => console.error("Fallback request failed:", e));
    }
    return true;
  }
});

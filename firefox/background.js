console.log('[Starchive] Background script loaded');

chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  console.log('[Starchive] Received message:', msg.type, msg);
  
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
    console.log(`[Starchive] Requesting txt for video ID: ${msg.videoId}`);
    fetch(`http://localhost:3009/get-txt?id=${msg.videoId}`)
      .then(res => {
        console.log(`[Starchive] Response status for ${msg.videoId}:`, res.status);
        return res.text();
      })
      .then(text => {
        console.log(`[Starchive] Got txt for ${msg.videoId}:`, text.substring(0, 100) + (text.length > 100 ? '...' : ''));
        
        // Check if the response contains actual transcript content or just a status message
        const hasContent = !text.includes('Download started for video') && !text.includes('already in download queue') && text.length > 50;
        
        sendResponse({
          hasContent: hasContent,
          content: hasContent ? text : null,
          videoId: msg.videoId
        });
      })
      .catch(err => {
        console.error(`[Starchive] Error getting txt for ${msg.videoId}:`, err);
        sendResponse({
          hasContent: false,
          error: err.message,
          videoId: msg.videoId
        });
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

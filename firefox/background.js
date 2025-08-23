console.log('[Starchive] Background script loaded');

// Store for transcript content
let storedTranscripts = {};

// Store for current mode
let currentMode = 'default';

// Function to collect all YouTube and Google authentication cookies
async function collectAllYouTubeCookies() {
  console.log(`[Starchive] 🚀 Starting comprehensive cookie collection for YouTube authentication`);
  
  const domains = [
    'youtube.com',
    '.youtube.com', 
    'www.youtube.com',
    'google.com',
    '.google.com',
    'www.google.com',
    'accounts.google.com',
    'apis.google.com',
    'play.google.com'
  ];
  
  const allCookies = [];
  const criticalCookieNames = [
    // YouTube specific cookies
    'VISITOR_INFO1_LIVE', 'VISITOR_PRIVACY_METADATA', 'PREF',
    'YSC', 'GPS', 'CONSENT',
    
    // Google authentication cookies  
    'SID', 'HSID', 'SSID', 'APISID', 'SAPISID',
    '__Secure-1PAPISID', '__Secure-3PAPISID', '__Secure-1PSID', '__Secure-3PSID',
    '__Secure-1PSIDTS', '__Secure-3PSIDTS', '__Secure-1PSIDCC', '__Secure-3PSIDCC',
    
    // Login and session cookies
    'LOGIN_INFO', 'session_logininfo', 'oauth_token',
    '__Host-1PLSID', '__Host-3PLSID', '__Host-GAPS',
    
    // Additional security cookies
    'NID', 'DV', '__Secure-ENID', '1P_JAR', 'AEC',
    'SMSV', 'ACCOUNT_CHOOSER', 'LSOLH'
  ];
  
  console.log(`[Starchive] 🔍 Searching ${domains.length} domains for cookies:`, domains);
  console.log(`[Starchive] 🎯 Looking for ${criticalCookieNames.length} critical authentication cookies:`, criticalCookieNames);
  
  for (let i = 0; i < domains.length; i++) {
    const domain = domains[i];
    console.log(`[Starchive] 🌐 [${i+1}/${domains.length}] Processing domain: ${domain}`);
    
    try {
      const cookies = await new Promise((resolve, reject) => {
        console.log(`[Starchive] 📡 Making browser.cookies.getAll request for domain: ${domain}`);
        browser.cookies.getAll({ domain: domain }, (cookies) => {
          if (browser.runtime.lastError) {
            console.error(`[Starchive] ❌ Error getting cookies for ${domain}:`, browser.runtime.lastError);
            resolve([]);
          } else {
            const cookieCount = (cookies || []).length;
            console.log(`[Starchive] ✅ Successfully retrieved ${cookieCount} cookies for domain: ${domain}`);
            resolve(cookies || []);
          }
        });
      });
      
      if (cookies.length === 0) {
        console.warn(`[Starchive] ⚠️  No cookies found for domain: ${domain}`);
        continue;
      }
      
      console.log(`[Starchive] 📋 Raw cookies for ${domain}:`);
      cookies.forEach((cookie, idx) => {
        const isCritical = criticalCookieNames.includes(cookie.name);
        const isSecure = cookie.name.startsWith('__Secure-') || cookie.name.startsWith('__Host-');
        const hasSession = cookie.name.includes('session') || cookie.name.includes('login') || cookie.name.includes('auth');
        
        console.log(`[Starchive]   [${idx+1}] ${isCritical ? '🔑' : isSecure ? '🔒' : hasSession ? '👤' : '🍪'} ${cookie.name} = ${cookie.value.substring(0, 20)}... (domain: ${cookie.domain}, secure: ${cookie.secure}, httpOnly: ${cookie.httpOnly})`);
      });
      
      // Filter to include all cookies, but log why each is included
      const filteredCookies = cookies.filter((cookie, idx) => {
        // Always include critical authentication cookies
        if (criticalCookieNames.includes(cookie.name)) {
          console.log(`[Starchive] 🔑 INCLUDING critical cookie [${idx+1}]: ${cookie.name} for ${domain}`);
          return true;
        }
        
        // Include session and security related cookies
        if (cookie.name.includes('session') || cookie.name.includes('login') || 
            cookie.name.includes('auth') || cookie.name.includes('token') ||
            cookie.name.startsWith('__Secure-') || cookie.name.startsWith('__Host-')) {
          console.log(`[Starchive] 🔒 INCLUDING security cookie [${idx+1}]: ${cookie.name} for ${domain}`);
          return true;
        }
        
        // Include all other cookies too (YouTube's bot detection might check any cookie)
        console.log(`[Starchive] 🍪 INCLUDING general cookie [${idx+1}]: ${cookie.name} for ${domain}`);
        return true;
      });
      
      console.log(`[Starchive] ✨ Domain ${domain}: ${cookies.length} total → ${filteredCookies.length} included`);
      allCookies.push(...filteredCookies);
      
    } catch (error) {
      console.error(`[Starchive] 💥 Failed to get cookies for domain ${domain}:`, error);
    }
  }
  
  console.log(`[Starchive] 📊 Raw collection complete: ${allCookies.length} cookies from all domains`);
  
  // Remove duplicates based on name and domain
  const uniqueCookies = allCookies.filter((cookie, index, self) => {
    const isDuplicate = index !== self.findIndex(c => c.name === cookie.name && c.domain === cookie.domain);
    if (isDuplicate) {
      console.log(`[Starchive] 🗑️  Removing duplicate: ${cookie.name}@${cookie.domain}`);
    }
    return !isDuplicate;
  });
  
  console.log(`[Starchive] 🎯 After deduplication: ${uniqueCookies.length} unique cookies`);
  
  // Log critical cookies found
  const foundCritical = uniqueCookies.filter(c => criticalCookieNames.includes(c.name));
  console.log(`[Starchive] 🔑 Found ${foundCritical.length}/${criticalCookieNames.length} critical authentication cookies:`);
  foundCritical.forEach((cookie, idx) => {
    console.log(`[Starchive]   [${idx+1}] ${cookie.name}@${cookie.domain} = ${cookie.value.substring(0, 30)}...`);
  });
  
  // Log missing critical cookies
  const missingCritical = criticalCookieNames.filter(name => !foundCritical.some(c => c.name === name));
  if (missingCritical.length > 0) {
    console.warn(`[Starchive] ⚠️  Missing ${missingCritical.length} critical cookies:`, missingCritical);
  }
  
  // Summarize by domain
  const domainSummary = {};
  uniqueCookies.forEach(cookie => {
    domainSummary[cookie.domain] = (domainSummary[cookie.domain] || 0) + 1;
  });
  console.log(`[Starchive] 📈 Cookies by domain:`, domainSummary);
  
  console.log(`[Starchive] 🏁 Cookie collection complete: ${uniqueCookies.length} total cookies ready for authentication`);
  return uniqueCookies;
}

browser.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  console.log('[Starchive] Received message:', msg.type, msg);
  
  if (msg.type === "fetchData") {
    console.log('[Starchive] Fetching data from /data endpoint');
    fetch("http://localhost:3009/data")
      .then(res => {
        console.log('[Starchive] Response status:', res.status);
        return res.json();
      })
      .then(data => {
        console.log('[Starchive] Received data from server:', data);
        console.log('[Starchive] diskUsage in response:', data.diskUsage);
        sendResponse(data);
      })
      .catch(err => {
        console.error('[Starchive] Error fetching data:', err);
        sendResponse({ error: err.message });
      });
    return true;
  }
  
  if (msg.type === "requestTxt") {
    console.log(`[Starchive] Requesting txt for video ID: ${msg.videoId}`);
    console.log(`[Starchive] sendResponse function available:`, typeof sendResponse);
    
    fetch(`http://localhost:3009/get-txt?id=${msg.videoId}&mode=${currentMode}`)
      .then(res => {
        console.log(`[Starchive] Response status for ${msg.videoId}:`, res.status);
        return res.json();
      })
      .then(data => {
        console.log(`[Starchive] Got response for ${msg.videoId}:`, data);
        
        console.log(`[Starchive] About to send response - hasContent: ${data.hasContent}, videoId: ${msg.videoId}`);
        
        const responseObj = {
          hasContent: data.hasContent,
          content: data.content || null,
          videoId: msg.videoId,
          message: data.message || null
        };
        
        console.log(`[Starchive] Sending response object:`, responseObj);
        sendResponse(responseObj);
        console.log(`[Starchive] Response sent for ${msg.videoId}`);
      })
      .catch(err => {
        console.error(`[Starchive] Error getting txt for ${msg.videoId}:`, err);
        const errorResponse = {
          hasContent: false,
          error: err.message,
          videoId: msg.videoId
        };
        console.log(`[Starchive] Sending error response:`, errorResponse);
        sendResponse(errorResponse);
        console.log(`[Starchive] Error response sent for ${msg.videoId}`);
      });
    return true;
  }
  
  if (msg.type === "storeTranscript") {
    console.log(`[Starchive] Storing transcript for ${msg.videoId}, length: ${msg.content.length}`);
    storedTranscripts[msg.videoId] = {
      content: msg.content,
      timestamp: Date.now()
    };
    console.log(`[Starchive] Stored transcripts count: ${Object.keys(storedTranscripts).length}`);
    return true;
  }
  
  if (msg.type === "getStoredTranscript") {
    console.log('[Starchive] Get stored transcript request from popup');
    
    if (Object.keys(storedTranscripts).length === 0) {
      console.log('[Starchive] No stored transcripts available');
      sendResponse({
        success: false,
        error: 'No transcript available. Hover over a YouTube video with a transcript first.'
      });
      return true;
    }
    
    // Get the most recently stored transcript
    const sortedTranscripts = Object.entries(storedTranscripts)
      .sort((a, b) => b[1].timestamp - a[1].timestamp);
    
    const [videoId, transcriptData] = sortedTranscripts[0];
    
    console.log(`[Starchive] Returning transcript for ${videoId}, length: ${transcriptData.content.length}`);
    
    sendResponse({
      success: true,
      videoId: videoId,
      content: transcriptData.content,
      length: transcriptData.content.length
    });
    
    return true;
  }
  
  if (msg.type === "setMode") {
    console.log(`[Starchive] Mode changed to: ${msg.mode}`);
    currentMode = msg.mode;
    return true;
  }
  
  if (msg.type === "youtubeVideo") {
    console.log(`[Starchive] 🎬 YouTube video detected: ${msg.videoId}`);
    console.log(`[Starchive] 🔄 Starting cookie collection and backend transmission process...`);
    
    // Collect all YouTube and Google cookies for authentication
    try {
      collectAllYouTubeCookies().then(allCookies => {
        console.log(`[Starchive] ✅ Cookie collection successful: ${allCookies.length} total cookies`);
        
        const minimalCookies = allCookies.map(c => ({
          name: c.name,
          value: c.value,
          domain: c.domain,
          path: c.path || "/",
          expires: c.expirationDate || 0,
          secure: !!c.secure,
          httpOnly: !!c.httpOnly
        }));

        console.log(`[Starchive] 📦 Preparing payload for backend:`);
        console.log(`[Starchive]   - Video ID: ${msg.videoId}`);
        console.log(`[Starchive]   - Cookies: ${minimalCookies.length} items`);
        console.log(`[Starchive]   - Payload size: ~${JSON.stringify({ videoId: msg.videoId, cookies: minimalCookies }).length} bytes`);
        
        // Log sample of cookies being sent
        const criticalInPayload = minimalCookies.filter(c => ['SAPISID', 'SID', 'HSID', '__Secure-3PAPISID', 'LOGIN_INFO', 'session_logininfo'].includes(c.name));
        console.log(`[Starchive] 🔑 Critical cookies in payload (${criticalInPayload.length}):`, criticalInPayload.map(c => `${c.name}@${c.domain}`));

        console.log(`[Starchive] 🌐 Sending POST request to http://localhost:3009/youtube`);
        fetch("http://localhost:3009/youtube", {
          method: "POST",
          headers: {
            "Content-Type": "application/json"
          },
          body: JSON.stringify({ videoId: msg.videoId, cookies: minimalCookies })
        })
          .then(res => {
            console.log(`[Starchive] 📥 Backend response received - Status: ${res.status} ${res.statusText}`);
            return res.text();
          })
          .then(text => {
            console.log(`[Starchive] 📄 Raw backend response: "${text.substring(0, 200)}${text.length > 200 ? '...' : ''}"`);
            // Try to parse JSON, but fall back to plain text
            let data;
            try { 
              data = JSON.parse(text);
              console.log(`[Starchive] ✅ Backend response parsed as JSON:`, data);
            } catch (_) { 
              data = { message: text };
              console.log(`[Starchive] 📝 Backend response treated as plain text`);
            }
            console.log(`[Starchive] 🎯 Final result for video ${msg.videoId}: ${minimalCookies.length} cookies sent, response received`);
            if (sendResponse) sendResponse(data);
          })
          .catch(err => {
            console.error(`[Starchive] ❌ Backend request failed for ${msg.videoId}:`, err);
            console.error(`[Starchive] 🔍 Error details:`, { 
              name: err.name, 
              message: err.message, 
              stack: err.stack?.substring(0, 200) 
            });
            if (sendResponse) sendResponse({ error: err.message });
          });
      }).catch(err => {
        console.error(`[Starchive] 💥 Cookie collection failed for ${msg.videoId}:`, err);
        console.log(`[Starchive] 🔄 Attempting fallback request without cookies...`);
        fetch("http://localhost:3009/youtube", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ videoId: msg.videoId })
        }).catch(e => console.error(`[Starchive] ❌ Fallback request also failed:`, e));
      });
    } catch (err) {
      console.error(`[Starchive] 💀 Critical error in YouTube handler for ${msg.videoId}:`, err);
      console.log(`[Starchive] 🔄 Attempting emergency fallback...`);
      fetch("http://localhost:3009/youtube", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ videoId: msg.videoId })
      }).catch(e => console.error(`[Starchive] ❌ Emergency fallback failed:`, e));
    }
    return true;
  }

  if (msg.type === "instagramPost") {
    // Collect Instagram cookies then post post + cookies to backend
    try {
      browser.cookies.getAll({ domain: "instagram.com" }, (cookies) => {
        if (browser.runtime.lastError) {
          console.error("Error getting Instagram cookies:", browser.runtime.lastError);
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

        fetch("http://localhost:3009/instagram", {
          method: "POST",
          headers: {
            "Content-Type": "application/json"
          },
          body: JSON.stringify({ postId: msg.postId, cookies: minimalCookies })
        })
          .then(res => res.text())
          .then(text => {
            // Try to parse JSON, but fall back to plain text
            let data;
            try { data = JSON.parse(text); } catch (_) { data = { message: text }; }
            console.log("Instagram post sent:", msg.postId, "Cookies:", minimalCookies.length, "Response:", data);
            if (sendResponse) sendResponse(data);
          })
          .catch(err => {
            console.error("Error sending Instagram post:", err);
            if (sendResponse) sendResponse({ error: err.message });
          });
      });
    } catch (err) {
      console.error("Instagram cookie collection failed:", err);
      fetch("http://localhost:3009/instagram", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ postId: msg.postId })
      }).catch(e => console.error("Instagram fallback request failed:", e));
    }
    return true;
  }
  
  if (msg.type === "sendPOToken") {
    console.log('[Starchive] Received PO token from content script:', msg.poToken.substring(0, 20) + '...');
    
    // Send PO token to backend
    fetch("http://localhost:3009/po-token", {
      method: "POST",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify({ 
        poToken: msg.poToken, 
        timestamp: msg.timestamp,
        source: 'extension'
      })
    })
    .then(res => res.json())
    .then(data => {
      console.log('[Starchive] PO token sent to backend:', data);
    })
    .catch(err => {
      console.error('[Starchive] Error sending PO token to backend:', err);
    });
    
    return true;
  }
});

// Handle toolbar button clicks
(browser.action || browser.browserAction).onClicked.addListener((tab) => {
  console.log('[Starchive] Toolbar button clicked, checking for stored transcripts');
  
  if (Object.keys(storedTranscripts).length === 0) {
    console.log('[Starchive] No stored transcripts available');
    browser.notifications.create({
      type: 'basic',
      iconUrl: 'icon.png',
      title: 'Starchive',
      message: 'No transcript available. Hover over a YouTube video with a transcript first.'
    });
    return;
  }
  
  // Get the most recently stored transcript
  const sortedTranscripts = Object.entries(storedTranscripts)
    .sort((a, b) => b[1].timestamp - a[1].timestamp);
  
  const [videoId, transcriptData] = sortedTranscripts[0];
  
  console.log(`[Starchive] Copying transcript for ${videoId} to clipboard, length: ${transcriptData.content.length}`);
  
  // Copy to clipboard
  navigator.clipboard.writeText(transcriptData.content).then(() => {
    console.log('[Starchive] Transcript copied to clipboard via toolbar button');
    browser.notifications.create({
      type: 'basic',
      iconUrl: 'icon.png',
      title: 'Starchive',
      message: `📋 Transcript for ${videoId} copied to clipboard! (${transcriptData.content.length} chars)`
    });
  }).catch(err => {
    console.error('[Starchive] Failed to copy via toolbar button:', err);
    browser.notifications.create({
      type: 'basic',
      iconUrl: 'icon.png', 
      title: 'Starchive',
      message: '❌ Failed to copy transcript to clipboard'
    });
  });
});

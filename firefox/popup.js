document.getElementById('fetchButton').addEventListener('click', () => {
  console.log('[Starchive] Fetch button clicked in popup');
  browser.runtime.sendMessage({ type: "fetchData" }, (response) => {
    console.log('[Starchive] Received response in popup:', response);
    const resultDiv = document.getElementById('result');
    if (response && response.error) {
      console.log('[Starchive] Error in response:', response.error);
      resultDiv.textContent = `Error: ${response.error}`;
    } else if (response) {
      console.log('[Starchive] Processing response, diskUsage:', response.diskUsage);
      // Display server status and disk usage in a user-friendly format
      let output = `Status: ${response.status}\n`;
      if (response.diskUsage) {
        console.log('[Starchive] Found diskUsage object:', response.diskUsage);
        if (response.diskUsage.error) {
          console.log('[Starchive] Disk usage error:', response.diskUsage.error);
          output += `Disk: ${response.diskUsage.error}\n`;
        } else {
          console.log('[Starchive] Adding disk usage to output');
          output += `Disk Usage for ./data:\n`;
          output += `  Total: ${response.diskUsage.totalPretty}\n`;
          output += `  Used:  ${response.diskUsage.usedPretty}\n`;
          output += `  Free:  ${response.diskUsage.freePretty}\n`;
        }
      } else {
        console.log('[Starchive] No diskUsage found in response');
        output += `No disk usage data available\n`;
      }
      console.log('[Starchive] Final output:', output);
      resultDiv.textContent = output;
    } else {
      console.log('[Starchive] No response received');
      resultDiv.textContent = 'No response received';
    }
  });
});

document.getElementById('copyTranscriptButton').addEventListener('click', () => {
  console.log('[Starchive] Copy transcript button clicked in popup');
  browser.runtime.sendMessage({ type: "getStoredTranscript" }, (response) => {
    const resultDiv = document.getElementById('result');
    
    if (response && response.success && response.content) {
      console.log(`[Starchive] Got transcript content, copying to clipboard: ${response.content.length} chars`);
      
      // Copy in popup context (has proper user activation)
      navigator.clipboard.writeText(response.content).then(() => {
        console.log('[Starchive] Transcript copied successfully from popup');
        resultDiv.textContent = `✅ Copied ${response.videoId} (${response.length} chars)`;
        resultDiv.style.background = '#d4edda';
        resultDiv.style.color = '#155724';
      }).catch(err => {
        console.error('[Starchive] Clipboard copy failed in popup:', err);
        // Fallback method
        const textArea = document.createElement('textarea');
        textArea.value = response.content;
        document.body.appendChild(textArea);
        textArea.select();
        try {
          document.execCommand('copy');
          resultDiv.textContent = `✅ Copied ${response.videoId} (${response.length} chars) [fallback]`;
          resultDiv.style.background = '#d4edda';
          resultDiv.style.color = '#155724';
        } catch (fallbackErr) {
          resultDiv.textContent = `❌ Failed to copy to clipboard`;
          resultDiv.style.background = '#f8d7da';
          resultDiv.style.color = '#721c24';
        }
        document.body.removeChild(textArea);
      });
      
    } else if (response && response.error) {
      resultDiv.textContent = `❌ ${response.error}`;
      resultDiv.style.background = '#f8d7da';
      resultDiv.style.color = '#721c24';
    } else {
      resultDiv.textContent = '❌ No response received';
      resultDiv.style.background = '#f8d7da';
      resultDiv.style.color = '#721c24';
    }
  });
});
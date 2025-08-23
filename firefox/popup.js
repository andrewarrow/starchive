function updateDiskUsageUI(diskUsage) {
  const container = document.getElementById('diskUsageContainer');
  const diskBarFill = document.getElementById('diskBarFill');
  const diskUsageStats = document.getElementById('diskUsageStats');
  const dataFolderSize = document.getElementById('dataFolderSize');
  const dataFolderBarFill = document.getElementById('dataFolderBarFill');
  const dataFolderPercent = document.getElementById('dataFolderPercent');

  if (diskUsage.error) {
    container.classList.add('visible');
    diskUsageStats.textContent = `Error: ${diskUsage.error}`;
    return;
  }

  // Calculate percentage of disk used
  const usedPercent = (diskUsage.used / diskUsage.total) * 100;
  
  // Show the container
  container.classList.add('visible');
  
  // Update disk usage bar
  diskBarFill.style.width = `${usedPercent}%`;
  
  // Update disk usage stats
  diskUsageStats.innerHTML = `
    Total: ${diskUsage.totalPretty} • Used: ${diskUsage.usedPretty} (${usedPercent.toFixed(1)}%) • Free: ${diskUsage.freePretty}
  `;
  
  // Update data folder info
  dataFolderSize.textContent = diskUsage.dataSizePretty || 'Unknown';
  
  if (diskUsage.dataPercentOfFree) {
    const dataPercent = Math.min(diskUsage.dataPercentOfFree, 100); // Cap at 100%
    dataFolderBarFill.style.width = `${dataPercent}%`;
    dataFolderPercent.textContent = `${diskUsage.dataPercentOfFree.toFixed(1)}% of free space`;
  } else {
    dataFolderBarFill.style.width = '0%';
    dataFolderPercent.textContent = 'Unable to calculate percentage';
  }
}

document.getElementById('fetchButton').addEventListener('click', () => {
  console.log('[Starchive] Disk icon clicked - toggling disk usage visibility');
  const container = document.getElementById('diskUsageContainer');
  
  // If already visible, just toggle it
  if (container.classList.contains('visible')) {
    container.classList.remove('visible');
    console.log('[Starchive] Hiding disk usage container');
    return;
  }
  
  // If not visible, fetch data and show it
  console.log('[Starchive] Fetching disk data to show usage');
  browser.runtime.sendMessage({ type: "fetchData" }, (response) => {
    console.log('[Starchive] Received response in popup:', response);
    const resultDiv = document.getElementById('result');
    
    if (response && response.error) {
      console.log('[Starchive] Error in response:', response.error);
      resultDiv.textContent = `Error: ${response.error}`;
      container.classList.remove('visible');
    } else if (response) {
      console.log('[Starchive] Processing response, diskUsage:', response.diskUsage);
      
      if (response.diskUsage) {
        console.log('[Starchive] Found diskUsage object:', response.diskUsage);
        updateDiskUsageUI(response.diskUsage);
        resultDiv.textContent = '';
      } else {
        console.log('[Starchive] No diskUsage found in response');
        resultDiv.textContent = `No disk usage data available`;
        container.classList.remove('visible');
      }
    } else {
      console.log('[Starchive] No response received');
      resultDiv.textContent = 'No response received';
      container.classList.remove('visible');
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

document.getElementById('podpapyrusButton').addEventListener('click', () => {
  console.log('[Starchive] Podpapyrus button clicked');
  const button = document.getElementById('podpapyrusButton');
  const modeText = document.getElementById('modeText');
  
  // Toggle pressed state
  if (button.classList.contains('pressed')) {
    button.classList.remove('pressed');
    modeText.textContent = 'Mode: default';
    console.log('[Starchive] Switched to default mode');
    
    // Send mode change to background script
    browser.runtime.sendMessage({ type: "setMode", mode: "default" });
  } else {
    button.classList.add('pressed');
    modeText.textContent = 'Mode: podpapyrus';
    console.log('[Starchive] Switched to podpapyrus mode');
    
    // Send mode change to background script
    browser.runtime.sendMessage({ type: "setMode", mode: "podpapyrus" });
  }
});
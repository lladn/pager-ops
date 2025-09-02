<script>
  import { onMount } from 'svelte';
  import Settings from './components/Settings.svelte';
  import IncidentsPanel from './components/IncidentsPanel.svelte';
  import { LogDebug, EventsEmit } from '../wailsjs/runtime/runtime.js';
  
  let showSettings = false;
  let runtimeReady = false;
  
  onMount(async () => {
    // Wait for DOM and Wails runtime to be ready
    const waitForRuntime = async () => {
      let attempts = 0;
      while (attempts < 50) {
        try {
          // Try to call a runtime function to test if it's ready
          await EventsEmit('app:ready');
          runtimeReady = true;
          LogDebug("App mounted successfully");
          return true;
        } catch (error) {
          attempts++;
          await new Promise(resolve => setTimeout(resolve, 50));
        }
      }
      console.error("Failed to initialize Wails runtime");
      return false;
    };
    
    const ready = await waitForRuntime();
    
    if (ready) {
      // Set up click handlers after runtime is ready
      document.addEventListener('click', (e) => {
        const target = e.target;
        if (target instanceof HTMLElement) {
          LogDebug(`Global click detected on: ${target.tagName}, class: ${target.className}`);
        }
      });
    }
  });
  
  function toggleSettings() {
    if (!runtimeReady) {
      console.log('Runtime not ready yet');
      return;
    }
    LogDebug(`Settings toggle clicked, current state: ${showSettings}`);
    console.log('Settings button clicked, current state:', showSettings);
    showSettings = !showSettings;
    LogDebug(`Settings new state: ${showSettings}`);
  }
</script>

<main style="--wails-draggable: no-drag;">
  <div class="app-header">
    <h4>PagerOps</h4>
    <button 
      class="settings-btn" 
      on:click|preventDefault|stopPropagation={toggleSettings}
      type="button"
      style="--wails-draggable: no-drag;">
      <svg 
        xmlns="http://www.w3.org/2000/svg" 
        width="15" 
        height="15" 
        viewBox="0 0 24 24" 
        fill="none" 
        stroke="currentColor" 
        stroke-width="2" 
        stroke-linecap="round" 
        stroke-linejoin="round">
        <circle cx="12" cy="12" r="3" />
        <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09c0-.67-.39-1.28-1-1.51a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06c.45-.45.58-1.12.33-1.82a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09c.67 0 1.28-.39 1.51-1a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06c.45.45 1.12.58 1.82.33h.01c.61-.23 1-.84 1-1.51V3a2 2 0 0 1 4 0v.09c0 .67.39 1.28 1 1.51.7.25 1.37.12 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06c-.45.45-.58 1.12-.33 1.82.23.61.84 1 1.51 1H21a2 2 0 0 1 0 4h-.09c-.67 0-1.28.39-1.51 1z"></path>
      </svg>
    </button>
  </div>
  
  <div class="app-content" style="--wails-draggable: no-drag;">
    <IncidentsPanel />
  </div>
  
  <Settings bind:isOpen={showSettings} />
</main>

<style>
  :global(body) {
    margin: 0;
    padding: 0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, 'Open Sans', 'Helvetica Neue', sans-serif;
  }

  :global(*) {
    box-sizing: border-box;
  }

  main {
    height: 100vh;
    display: flex;
    flex-direction: column;
    background: #1a1a1a;
    overflow: hidden;
  }

  .app-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 15px 20px;
    background: #111;
    border-bottom: 1px solid #333;
    position: relative;
    z-index: 10;
  }

  .app-header h4 {
    margin: 0;
    color: #fff;
    font-size: 20px;
    user-select: none;
  }

  .settings-btn {
    background: none;
    border: none;
    color: #999;
    cursor: pointer;
    padding: 8px;
    border-radius: 4px;
    transition: all 0.3s;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .settings-btn:hover {
    background: #333;
    color: #fff;
  }

  .settings-btn:active {
    transform: scale(0.95);
  }

  .app-content {
    flex: 1;
    overflow: hidden;
    position: relative;
  }
</style>
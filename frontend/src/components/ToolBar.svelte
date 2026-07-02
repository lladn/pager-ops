<script lang="ts">
    import { settingsOpen, panelOpen } from '../stores/incidents';
    import { theme, toggleTheme } from '../stores/theme';
    import { SetTheme } from '../../wailsjs/go/main/App';
    
    function openSettings() {
        settingsOpen.set(true);
    }
    
    function togglePanel() {
        panelOpen.update(value => !value);
    }
    
    async function handleToggleTheme() {
        toggleTheme();
        try {
            await SetTheme($theme);
        } catch (err) {
            console.error('Failed to persist theme setting:', err);
        }
    }
</script>

<div class="toolbar">
    <div class="toolbar-content">
        <div class="toolbar-left">
            <!-- Space for macOS traffic lights -->
            <div class="traffic-light-spacer"></div>
            <h3 class="app-title">PagerOps</h3>
        </div>
        <div class="toolbar-right">
            <button class="theme-button" on:click={handleToggleTheme} title="Toggle theme">
                {#if $theme === 'dark'}
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <circle cx="12" cy="12" r="5"></circle>
                        <line x1="12" y1="1" x2="12" y2="3"></line>
                        <line x1="12" y1="21" x2="12" y2="23"></line>
                        <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line>
                        <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line>
                        <line x1="1" y1="12" x2="3" y2="12"></line>
                        <line x1="21" y1="12" x2="23" y2="12"></line>
                        <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line>
                        <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>
                    </svg>
                {:else}
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
                    </svg>
                {/if}
            </button>
            <button class="panel-button" on:click={togglePanel} title="Toggle Panel" class:active={$panelOpen}>
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
                    <line x1="15" y1="3" x2="15" y2="21"></line>
                    <polyline points="10 9 7 12 10 15"></polyline>
                </svg>
            </button>
            <button class="settings-button" on:click={openSettings} title="Settings">
                <svg width="18" height="18" viewBox="0 0 20 20" fill="currentColor">
                    <path fill-rule="evenodd" d="M11.49 3.17c-.38-1.56-2.6-1.56-2.98 0a1.532 1.532 0 01-2.286.948c-1.372-.836-2.942.734-2.106 2.106.54.886.061 2.042-.947 2.287-1.561.379-1.561 2.6 0 2.978a1.532 1.532 0 01.947 2.287c-.836 1.372.734 2.942 2.106 2.106a1.532 1.532 0 012.287.947c.379 1.561 2.6 1.561 2.978 0a1.533 1.533 0 012.287-.947c1.372.836 2.942-.734 2.106-2.106a1.533 1.533 0 01.947-2.287c1.561-.379 1.561-2.6 0-2.978a1.532 1.532 0 01-.947-2.287c.836-1.372-.734-2.942-2.106-2.106a1.532 1.532 0 01-2.287-.947zM10 13a3 3 0 100-6 3 3 0 000 6z" clip-rule="evenodd" />
                </svg>
            </button>
        </div>
    </div>
</div>

<style>
    .toolbar {
        background: linear-gradient(to bottom, var(--bg-primary), var(--bg-secondary));
        border-bottom: 1px solid var(--border);
        height: 38px;
        -webkit-app-region: drag;
        --wails-draggable: drag;
        cursor: default;
        /* Prevent accidental text highlight */
        -webkit-user-select: none;
        -ms-user-select: none;
        user-select: none;
    }
    
    .toolbar-content {
        display: flex;
        justify-content: space-between;
        align-items: center;
        height: 100%;
        padding: 0 12px;
    }
    
    .toolbar-left {
        display: flex;
        align-items: center;
    }
    
    .traffic-light-spacer {
        width: 70px;
        height: 100%;
    }
    
    .app-title {
        font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
        font-size: 15px;
        font-weight: 600;
        color: var(--text-primary);
        margin: 0;
        letter-spacing: -0.2px;
        white-space: nowrap;
    }
    
    .toolbar-right {
        display: flex;
        align-items: center;
        gap: 4px;
    }
    
    .theme-button,
    .panel-button,
    .settings-button {
        background: transparent;
        border: none;
        padding: 4px;
        border-radius: 4px;
        cursor: pointer;
        color: var(--text-tertiary);
        display: flex;
        align-items: center;
        justify-content: center;
        -webkit-app-region: no-drag;
        --wails-draggable: no-drag;
        transition: all 0.2s;
    }
    
    .theme-button:hover,
    .panel-button:hover,
    .settings-button:hover {
        background: var(--bg-active);
        color: var(--text-primary);
    }
    
    .panel-button.active {
        background: var(--accent-soft);
        color: var(--accent);
    }
</style>
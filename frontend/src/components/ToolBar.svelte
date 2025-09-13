<script lang="ts">
    import { settingsOpen } from '../stores/incidents';
    
    function openSettings() {
        settingsOpen.set(true);
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
        background: linear-gradient(to bottom, #fafafa, #f5f5f5);
        border-bottom: 1px solid #e0e0e0;
        height: 38px;
        -webkit-app-region: drag;
        --wails-draggable: drag;
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
        font-size: 14px;
        font-weight: 500;
        color: #2c2c2c;
        margin: 0;
        letter-spacing: -0.2px;
    }
    
    .toolbar-right {
        display: flex;
        align-items: center;
    }
    
    .settings-button {
        background: transparent;
        border: none;
        padding: 4px;
        border-radius: 4px;
        cursor: pointer;
        color: #666;
        display: flex;
        align-items: center;
        justify-content: center;
        -webkit-app-region: no-drag;
        --wails-draggable: no-drag;
    }
    
    .settings-button:hover {
        background: rgba(0, 0, 0, 0.06);
        color: #333;
    }
</style>
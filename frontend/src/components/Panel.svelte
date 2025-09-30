<script lang="ts">
    import { panelOpen, panelWidth } from '../stores/incidents';
    
    const MIN_WIDTH = 280;
    const MAX_WIDTH = 600;
    
    let isResizing = false;
    let startX = 0;
    let startWidth = 0;
    
    function startResize(event: MouseEvent) {
        isResizing = true;
        startX = event.clientX;
        startWidth = $panelWidth;
        
        document.body.style.cursor = 'ew-resize';
        document.body.style.userSelect = 'none';
        
        document.addEventListener('mousemove', handleResize);
        document.addEventListener('mouseup', stopResize);
    }
    
    function handleResize(event: MouseEvent) {
        if (!isResizing) return;
        
        const delta = startX - event.clientX;
        const newWidth = Math.max(MIN_WIDTH, Math.min(MAX_WIDTH, startWidth + delta));
        
        panelWidth.set(newWidth);
    }
    
    function stopResize() {
        isResizing = false;
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
        
        document.removeEventListener('mousemove', handleResize);
        document.removeEventListener('mouseup', stopResize);
    }
</script>

{#if $panelOpen}
    <div class="panel-container" style="width: {$panelWidth}px;">
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <div class="resize-handle" on:mousedown={startResize}></div>
        
        <div class="panel-header">
            <h5>Alert Details</h5>
            <button class="close-button" on:click={() => panelOpen.set(false)} title="Close Panel">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="18" y1="6" x2="6" y2="18"></line>
                    <line x1="6" y1="6" x2="18" y2="18"></line>
                </svg>
            </button>
        </div>
        <div class="panel-content">
            <!-- Panel content will go here -->
        </div>
    </div>
{/if}

<style>
    .panel-container {
        height: 100%;
        background: white;
        border-left: 1px solid #e0e0e0;
        display: flex;
        flex-direction: column;
        flex-shrink: 0;
        position: relative;
        min-width: 280px;
        max-width: 600px;
    }
    
    .resize-handle {
        position: absolute;
        left: 0;
        top: 0;
        bottom: 0;
        width: 4px;
        cursor: ew-resize;
        background: transparent;
        z-index: 10;
        transition: background 0.2s;
    }
    
    .resize-handle:hover {
        background: rgba(59, 130, 246, 0.3);
    }
    
    .resize-handle:active {
        background: rgba(59, 130, 246, 0.5);
    }
    
    .panel-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 14px;
        border-bottom: 1px solid #e0e0e0;
        background: #fafafa;
    }
    
    .panel-header h5 {
        margin: 0;
        font-size: 14px;
        font-weight: 600;
        color: #2c2c2c;
    }
    
    .close-button {
        background: transparent;
        border: none;
        padding: 4px;
        border-radius: 4px;
        cursor: pointer;
        color: #666;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: all 0.2s;
    }
    
    .close-button:hover {
        background: rgba(0, 0, 0, 0.06);
        color: #333;
    }
    
    .panel-content {
        flex: 1;
        overflow-y: auto;
        padding: 16px;
    }
    
    /* Custom scrollbar */
    .panel-content::-webkit-scrollbar {
        width: 8px;
    }
    
    .panel-content::-webkit-scrollbar-track {
        background: #f3f4f6;
    }
    
    .panel-content::-webkit-scrollbar-thumb {
        background: #d1d5db;
        border-radius: 4px;
    }
    
    .panel-content::-webkit-scrollbar-thumb:hover {
        background: #9ca3af;
    }
</style>
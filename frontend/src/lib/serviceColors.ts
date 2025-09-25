const SERVICE_COLORS = [
    '#1f77b4', // Strong Blue
    '#ff7f0e', // Vivid Orange
    '#2ca02c', // Bright Green
    '#d62728', // Strong Red
    '#9467bd', // Vibrant Purple
    '#8c564b', // Earthy Brown
    '#e377c2', // Bright Pink
    '#7f7f7f', // Neutral Dark Gray
    '#bcbd22', // Sharp Olive
    '#17becf', // Bright Cyan
    '#0d3b66', // Navy Blue
    '#e63946', // Crimson Red
    '#1d3557', // Deep Indigo Blue
    '#2a9d8f', // Teal Green
    '#f4a261', // Burnt Orange
    '#e9c46a', // Golden Yellow
    '#264653', // Slate Teal
    '#6a4c93', // Royal Purple
    '#ff006e', // Neon Pink
    '#8338ec', // Electric Violet
];



// Simple hash function to consistently map strings to indices
function hashString(str: string): number {
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
        const char = str.charCodeAt(i);
        hash = ((hash << 5) - hash) + char;
        hash = hash & hash; // Convert to 32-bit integer
    }
    return Math.abs(hash);
}

// Get color for a service name
export function getServiceColor(serviceName: string): string {
    if (!serviceName) return '#808080'; // Default gray for unknown
    
    const hash = hashString(serviceName.toLowerCase());
    const index = hash % SERVICE_COLORS.length;
    return SERVICE_COLORS[index];
}
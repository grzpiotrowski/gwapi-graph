class GatewayGraphVisualizer {
    constructor() {
        console.log('Initializing GatewayGraphVisualizer...');
        this.svg = null;
        this.width = 0;
        this.height = 0;
        this.simulation = null;
        this.nodes = [];
        this.links = [];
        this.selectedNode = null;
        this.autoRefresh = false;
        this.refreshInterval = null;
        this.websocket = null;
        this.zoom = null;
        this.layout = 'force';
        
        this.init();
        console.log('GatewayGraphVisualizer initialized');
    }

    init() {
        this.setupSVG();
        this.setupEventListeners();
        this.setupWebSocket();
        this.loadData();
    }

    setupSVG() {
        const container = document.getElementById('graph-container');
        this.width = container.clientWidth;
        this.height = container.clientHeight;

        this.svg = d3.select('#graph')
            .attr('width', this.width)
            .attr('height', this.height);

        // Create zoom behavior
        this.zoom = d3.zoom()
            .scaleExtent([0.1, 10])
            .on('zoom', (event) => {
                this.svg.select('.graph-group')
                    .attr('transform', event.transform);
            });

        this.svg.call(this.zoom);

        // Create main group for graph elements
        this.svg.append('g')
            .attr('class', 'graph-group');

        // Handle window resize
        window.addEventListener('resize', () => {
            this.width = container.clientWidth;
            this.height = container.clientHeight;
            this.svg.attr('width', this.width).attr('height', this.height);
            if (this.simulation) {
                this.simulation.force('center', d3.forceCenter(this.width / 2, this.height / 2));
                this.simulation.alpha(0.3).restart();
            }
        });
    }

    setupEventListeners() {
        // Refresh button
        document.getElementById('refresh-btn').addEventListener('click', () => {
            this.loadData();
        });

        // Auto-refresh toggle
        document.getElementById('auto-refresh-btn').addEventListener('click', () => {
            this.toggleAutoRefresh();
        });

        // Reset zoom button
        document.getElementById('reset-zoom-btn').addEventListener('click', () => {
            this.resetZoom();
        });

        // Layout selector
        document.getElementById('layout-select').addEventListener('change', (e) => {
            this.layout = e.target.value;
            this.updateLayout();
        });
    }

    setupWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/api/ws`;
        
        this.websocket = new WebSocket(wsUrl);
        
        this.websocket.onmessage = (event) => {
            const data = JSON.parse(event.data);
            this.updateGraph(data);
        };

        this.websocket.onclose = () => {
            console.log('WebSocket connection closed');
            // Attempt to reconnect after 5 seconds
            setTimeout(() => {
                this.setupWebSocket();
            }, 5000);
        };

        this.websocket.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
    }

    async loadData() {
        console.log('Loading data from /api/graph...');
        try {
            const response = await fetch('/api/graph');
            console.log('Response status:', response.status);
            const data = await response.json();
            console.log('Received data:', data);
            this.updateGraph(data);
        } catch (error) {
            console.error('Error loading data:', error);
        }
    }

    updateGraph(data) {
        console.log('Updating graph with data:', data);
        
        // Store the old node positions
        const oldNodesMap = new Map();
        this.nodes.forEach(node => {
            oldNodesMap.set(node.id, { x: node.x, y: node.y, fx: node.fx, fy: node.fy });
        });
        
        this.nodes = data.nodes || [];
        this.links = data.links || [];
        
        // Preserve positions for existing nodes
        this.nodes.forEach(node => {
            const oldPos = oldNodesMap.get(node.id);
            if (oldPos) {
                node.x = oldPos.x;
                node.y = oldPos.y;
                node.fx = oldPos.fx;
                node.fy = oldPos.fy;
            }
        });
        
        console.log(`Graph update: ${this.nodes.length} nodes, ${this.links.length} links`);
        console.log('Nodes:', this.nodes);
        console.log('Links:', this.links);

        // Update simulation only if needed
        this.updateSimulation();
        
        // Render the graph
        this.render();
    }

    updateSimulation() {
        const isFirstRun = !this.simulation;
        
        if (this.simulation) {
            // Update existing simulation with new data
            this.simulation.nodes(this.nodes);
            
            // Update force link with new links
            const linkForce = this.simulation.force('link');
            if (linkForce) {
                linkForce.links(this.links);
            }
            
            // Only restart if there are new nodes or significant changes
            if (isFirstRun) {
                this.simulation.alpha(0.3).restart();
            } else {
                // Gentle restart to accommodate new nodes
                this.simulation.alpha(0.1).restart();
            }
        } else {
            // Create new simulation
            this.simulation = d3.forceSimulation(this.nodes);
            this.setupLayoutForces();
        }
    }

    setupLayoutForces() {
        switch (this.layout) {
            case 'force':
                this.setupForceLayout();
                break;
            case 'radial':
                this.setupRadialLayout();
                break;
            case 'hierarchical':
                this.setupHierarchicalLayout();
                break;
        }
    }

    setupForceLayout() {
        this.simulation
            .force('link', d3.forceLink(this.links).distance(100))
            .force('charge', d3.forceManyBody().strength(-300))
            .force('center', d3.forceCenter(this.width / 2, this.height / 2))
            .force('collision', d3.forceCollide().radius(30));
    }

    setupRadialLayout() {
        // Find gateway classes as central nodes
        const gatewayClasses = this.nodes.filter(n => n.type === 'GatewayClass');
        const centerX = this.width / 2;
        const centerY = this.height / 2;
        
        this.simulation
            .force('link', d3.forceLink(this.links).distance(80))
            .force('charge', d3.forceManyBody().strength(-200))
            .force('center', d3.forceCenter(centerX, centerY))
            .force('radial', d3.forceRadial(d => {
                if (d.type === 'GatewayClass') return 0;
                if (d.type === 'Gateway') return 100;
                if (d.type === 'HTTPRoute') return 200;
                if (d.type === 'DNSRecord') return 300;
                if (d.type === 'Service') return 400;
                return 350;
            }, centerX, centerY))
            .force('collision', d3.forceCollide().radius(25));
    }

    setupHierarchicalLayout() {
        // Create a hierarchy based on resource relationships
        const hierarchy = this.createHierarchy();
        
        this.simulation
            .force('link', d3.forceLink(this.links).distance(60))
            .force('charge', d3.forceManyBody().strength(-150))
            .force('y', d3.forceY(d => d.hierarchyLevel * 120 + 50).strength(0.8))
            .force('x', d3.forceX(this.width / 2).strength(0.1))
            .force('collision', d3.forceCollide().radius(25));
    }

    createHierarchy() {
        // Assign hierarchy levels based on resource types
        this.nodes.forEach(node => {
            switch (node.type) {
                case 'GatewayClass':
                    node.hierarchyLevel = 0;
                    break;
                case 'Gateway':
                    node.hierarchyLevel = 1;
                    break;
                case 'Listener':
                    node.hierarchyLevel = 1.2; // Between Gateway and HTTPRoute
                    break;
                case 'HTTPRoute':
                    node.hierarchyLevel = 2;
                    break;
                case 'DNSRecord':
                    node.hierarchyLevel = 1.5; // Between Gateway and HTTPRoute
                    break;
                case 'Service':
                    node.hierarchyLevel = 3; // After HTTPRoute
                    break;
                case 'ReferenceGrant':
                    node.hierarchyLevel = 3;
                    break;
                default:
                    node.hierarchyLevel = 4;
            }
        });
    }

    render() {
        console.log('Starting render process...');
        console.log(`SVG dimensions: ${this.width}x${this.height}`);
        console.log(`Rendering ${this.nodes.length} nodes and ${this.links.length} links`);
        
        const g = this.svg.select('.graph-group');

        // Use D3 data join pattern for smooth updates
        this.renderLinks(g);
        this.renderNodes(g);

        // Update positions on simulation tick
        this.simulation.on('tick', () => {
            g.selectAll('.link')
                .attr('x1', d => d.source.x)
                .attr('y1', d => d.source.y)
                .attr('x2', d => d.target.x)
                .attr('y2', d => d.target.y);

            g.selectAll('.node-group')
                .attr('transform', d => `translate(${d.x},${d.y})`);
        });
    }

    renderLinks(g) {
        // Bind data to links
        const linkSelection = g.selectAll('.links')
            .data([0]); // Dummy data to ensure container exists

        const linksContainer = linkSelection.enter()
            .append('g')
            .attr('class', 'links')
            .merge(linkSelection);

        // Bind actual link data
        const links = linksContainer.selectAll('.link')
            .data(this.links, d => `${d.source.id || d.source}-${d.target.id || d.target}-${d.type}`);

        // Remove old links
        links.exit()
            .transition()
            .duration(300)
            .style('opacity', 0)
            .remove();

        // Add new links
        const newLinks = links.enter()
            .append('line')
            .attr('class', d => `link ${d.type}`)
            .style('opacity', 0)
            .on('mouseover', (event, d) => this.showTooltip(event, `${d.type} connection`))
            .on('mouseout', () => this.hideTooltip());

        // Update all links
        newLinks.merge(links)
            .transition()
            .duration(300)
            .style('opacity', 1);
    }

    renderNodes(g) {
        // Bind data to nodes
        const nodeSelection = g.selectAll('.nodes')
            .data([0]); // Dummy data to ensure container exists

        const nodesContainer = nodeSelection.enter()
            .append('g')
            .attr('class', 'nodes')
            .merge(nodeSelection);

        // Filter out hidden nodes unless they should be shown
        const visibleNodes = this.nodes.filter(node => !node.hidden);
        
        // Bind actual node data
        const nodes = nodesContainer.selectAll('.node-group')
            .data(visibleNodes, d => d.id);

        // Remove old nodes
        const exitingNodes = nodes.exit();
        exitingNodes
            .transition()
            .duration(300)
            .style('opacity', 0)
            .remove();

        // Add new nodes
        const newNodes = nodes.enter()
            .append('g')
            .attr('class', 'node-group')
            .style('opacity', 0)
            .call(d3.drag()
                .on('start', (event, d) => this.dragstarted(event, d))
                .on('drag', (event, d) => this.dragged(event, d))
                .on('end', (event, d) => this.dragended(event, d)));

        // Add circles for new nodes
        newNodes.append('circle')
            .attr('r', d => this.getNodeRadius(d))
            .attr('class', d => `node ${d.type.toLowerCase()}`)
            .on('click', (event, d) => this.handleNodeClick(event, d))
            .on('mouseover', (event, d) => this.showTooltip(event, this.getNodeTooltip(d)))
            .on('mouseout', () => this.hideTooltip());

        // Add labels for new nodes
        newNodes.append('text')
            .attr('class', 'node-label')
            .attr('dy', d => this.getNodeRadius(d) + 15)
            .text(d => d.name);

        // Add namespace labels for new nodes
        newNodes.append('text')
            .attr('class', 'namespace-label')
            .attr('dy', d => this.getNodeRadius(d) + 28)
            .text(d => d.namespace ? `(${d.namespace})` : '');

        // Update all nodes (new and existing)
        const allNodes = newNodes.merge(nodes);
        
        allNodes.transition()
            .duration(300)
            .style('opacity', 1);

        // Update existing node properties that might have changed
        allNodes.select('circle')
            .attr('r', d => this.getNodeRadius(d))
            .attr('class', d => `node ${d.type.toLowerCase()}`);

        allNodes.select('.node-label')
            .attr('dy', d => this.getNodeRadius(d) + 15)
            .text(d => d.name);

        allNodes.select('.namespace-label')
            .attr('dy', d => this.getNodeRadius(d) + 28)
            .text(d => d.namespace ? `(${d.namespace})` : '');
    }

    getNodeRadius(d) {
        const baseRadius = 12;
        const typeMultipliers = {
            'GatewayClass': 1.5,
            'Gateway': 1.3,
            'HTTPRoute': 1.0,
            'DNSRecord': 0.9,
            'Service': 1.1,
            'ReferenceGrant': 0.8,
            'Listener': 0.7
        };
        return baseRadius * (typeMultipliers[d.type] || 1.0);
    }

    getNodeTooltip(d) {
        if (d.type === 'Listener' && d.listenerData) {
            return `${d.type}: ${d.name} (Port ${d.listenerData.port}, ${d.listenerData.protocol}${d.listenerData.hostname ? `, ${d.listenerData.hostname}` : ''})`;
        }
        return `${d.type}: ${d.name}${d.namespace ? ` (${d.namespace})` : ''}`;
    }

    handleNodeClick(event, node) {
        // Standard node selection (removed gateway listener toggling)
        this.selectNode(node);
    }

    toggleGatewayListeners(gateway) {
        // Find all listener nodes for this gateway
        const listenerNodes = this.nodes.filter(node => 
            node.type === 'Listener' && node.parentId === gateway.id
        );

        if (listenerNodes.length === 0) return;

        // Toggle visibility of listener nodes
        const shouldShow = listenerNodes[0].hidden;
        listenerNodes.forEach(listener => {
            listener.hidden = !shouldShow;
        });

        // Re-render the graph to show/hide listeners
        this.render();
        
        // Restart simulation with gentle alpha to animate new nodes
        if (shouldShow) {
            this.simulation.alpha(0.2).restart();
        }
    }

    selectNode(node) {
        // Remove previous selection
        this.svg.selectAll('.node').classed('selected', false);
        
        // Select current node
        this.svg.selectAll('.node')
            .filter(d => d.id === node.id)
            .classed('selected', true);

        this.selectedNode = node;
        this.updateInfoPanel(node);
    }

    updateInfoPanel(node) {
        const infoContent = document.getElementById('info-content');
        
        if (!node) {
            infoContent.innerHTML = '<p>Click on a node to see resource details</p>';
            return;
        }

        let html = `
            <h4>${node.type}</h4>
            <div class="key-value">
                <span class="key">Name:</span>
                <span class="value">${node.name}</span>
            </div>
            <div class="key-value">
                <span class="key">Namespace:</span>
                <span class="value">${node.namespace || 'cluster-scoped'}</span>
            </div>
            <div class="key-value">
                <span class="key">Kind:</span>
                <span class="value">${node.kind}</span>
            </div>
            <div class="key-value">
                <span class="key">Group:</span>
                <span class="value">${node.group}</span>
            </div>
            <div class="key-value">
                <span class="key">Version:</span>
                <span class="value">${node.version}</span>
            </div>
        `;

        // Add listener-specific information
        if (node.type === 'Listener' && node.listenerData) {
            html += `
                <div class="key-value">
                    <span class="key">Port:</span>
                    <span class="value">${node.listenerData.port}</span>
                </div>
                <div class="key-value">
                    <span class="key">Protocol:</span>
                    <span class="value">${node.listenerData.protocol}</span>
                </div>
            `;
            
            if (node.listenerData.hostname) {
                html += `
                    <div class="key-value">
                        <span class="key">Hostname:</span>
                        <span class="value">${node.listenerData.hostname}</span>
                    </div>
                `;
            }
            
            html += `
                <div class="key-value">
                    <span class="key">TLS:</span>
                    <span class="value">${node.listenerData.tls ? 'Yes' : 'No'}</span>
                </div>
            `;
        }

        // Add gateway-specific information for gateways with listeners
        if (node.type === 'Gateway') {
            const listenerNodes = this.nodes.filter(n => 
                n.type === 'Listener' && n.parentId === node.id
            );
            if (listenerNodes.length > 0) {
                html += `
                    <div class="key-value">
                        <span class="key">Listeners:</span>
                        <span class="value">${listenerNodes.length}</span>
                    </div>
                `;
            }
        }

        html += `
            <div class="key-value">
                <span class="key">ID:</span>
                <span class="value">${node.id}</span>
            </div>
        `;
        
        infoContent.innerHTML = html;
    }

    showTooltip(event, text) {
        const tooltip = document.getElementById('tooltip');
        tooltip.style.left = event.pageX + 10 + 'px';
        tooltip.style.top = event.pageY - 10 + 'px';
        tooltip.textContent = text;
        tooltip.style.opacity = 1;
    }

    hideTooltip() {
        document.getElementById('tooltip').style.opacity = 0;
    }

    dragstarted(event, d) {
        if (!event.active) this.simulation.alphaTarget(0.3).restart();
        d.fx = d.x;
        d.fy = d.y;
    }

    dragged(event, d) {
        d.fx = event.x;
        d.fy = event.y;
    }

    dragended(event, d) {
        if (!event.active) this.simulation.alphaTarget(0);
        d.fx = null;
        d.fy = null;
    }

    toggleAutoRefresh() {
        const button = document.getElementById('auto-refresh-btn');
        this.autoRefresh = !this.autoRefresh;
        
        if (this.autoRefresh) {
            button.textContent = 'Auto Refresh: ON';
            button.classList.add('auto-refresh-on');
            this.refreshInterval = setInterval(() => {
                this.loadData();
            }, 10000); // Refresh every 10 seconds
        } else {
            button.textContent = 'Auto Refresh: OFF';
            button.classList.remove('auto-refresh-on');
            if (this.refreshInterval) {
                clearInterval(this.refreshInterval);
                this.refreshInterval = null;
            }
        }
    }

    resetZoom() {
        this.svg.transition().duration(750).call(
            this.zoom.transform,
            d3.zoomIdentity
        );
    }

    updateLayout() {
        // Update the simulation forces for the new layout
        if (this.simulation) {
            this.setupLayoutForces();
            // Give it a moderate restart to transition to new layout
            this.simulation.alpha(0.5).restart();
        }
    }
}

// Initialize the application when the DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    console.log('DOM loaded, initializing Gateway Graph Visualizer...');
    new GatewayGraphVisualizer();
}); 
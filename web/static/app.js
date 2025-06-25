class GatewayGraphVisualizer {
    constructor() {
        console.log('Initializing GatewayGraphVisualizer...');
        this.svg = null;
        this.width = 0;
        this.height = 0;
        this.simulation = null;
        this.nodes = [];
        this.links = [];
        this.dnsZones = [];
        this.selectedNode = null;
        this.autoRefresh = false;
        this.refreshInterval = null;
        this.websocket = null;
        this.zoom = null;
        this.layout = 'force';
        this.showDNSZones = true;
        
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

        // Add click handler to clear selections when clicking on empty space
        this.svg.on('click', (event) => {
            // Only clear selection if clicking directly on the SVG (not on any child elements)
            if (event.target === event.currentTarget) {
                this.clearSelection();
            }
        });

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

        // DNS zones toggle
        document.getElementById('dns-zones-toggle-btn').addEventListener('click', () => {
            this.toggleDNSZones();
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
        this.dnsZones = data.dnsZones || [];
        
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
        
        console.log(`Graph update: ${this.nodes.length} nodes, ${this.links.length} links, ${this.dnsZones.length} DNS zones`);
        console.log('Nodes:', this.nodes);
        console.log('Links:', this.links);
        console.log('DNS Zones:', this.dnsZones);

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
        console.log(`Rendering ${this.dnsZones.length} DNS zones, ${this.nodes.length} nodes and ${this.links.length} links`);
        
        const g = this.svg.select('.graph-group');

        // Use D3 data join pattern for smooth updates
        if (this.showDNSZones) {
            this.renderDNSZones(g);
        } else {
            // Remove DNS zones when disabled
            g.selectAll('.dns-zones').remove();
        }
        this.renderLinks(g);
        this.renderNodes(g);

        // Update positions on simulation tick
        this.simulation.on('tick', () => {
            // Update DNS zone hulls only if enabled
            if (this.showDNSZones) {
                this.updateDNSZones(g);
            }
            
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

    renderDNSZones(g) {
        // Filter out zones with no visible nodes and sort by hierarchy
        const visibleZones = this.dnsZones.filter(zone => {
            const zoneNodes = zone.nodes.map(nodeId => 
                this.nodes.find(n => n.id === nodeId)
            ).filter(Boolean);
            
            console.log(`Zone ${zone.name}: found ${zoneNodes.length} nodes out of ${zone.nodes.length} total`);
            
            const visibleNodes = zoneNodes.filter(n => {
                // For now, just check if node exists and isn't explicitly hidden
                // DOM elements might not be rendered yet when this is called
                const isVisible = n && !n.hidden;
                if (!isVisible) {
                    console.log(`Node ${n.id} (${n.name}) is not visible: exists=${!!n}, hidden=${n ? n.hidden : 'N/A'}`);
                }
                return isVisible;
            });
            
            console.log(`Zone ${zone.name}: ${visibleNodes.length} visible nodes out of ${zoneNodes.length}`);
            return visibleNodes.length > 0;
        });
        
        console.log(`Rendering ${this.dnsZones.length} total DNS zones, ${visibleZones.length} visible:`, visibleZones.map(z => z.name));
        
        // Sort zones by hierarchy (broader zones first, so they render behind more specific zones)
        const sortedZones = visibleZones.sort((a, b) => {
            const aDepth = a.name.split('.').length;
            const bDepth = b.name.split('.').length;
            return aDepth - bDepth; // Broader zones (fewer dots) first
        });
        
        // Create DNS zones container
        const zonesSelection = g.selectAll('.dns-zones')
            .data([0]); // Dummy data to ensure container exists

        const zonesContainer = zonesSelection.enter()
            .append('g')
            .attr('class', 'dns-zones')
            .merge(zonesSelection);

        // Bind DNS zone data (sorted by hierarchy)
        const zones = zonesContainer.selectAll('.dns-zone')
            .data(sortedZones, d => d.name);

        // Remove old zones
        zones.exit()
            .transition()
            .duration(300)
            .style('opacity', 0)
            .remove();

        // Add new zones
        const newZones = zones.enter()
            .append('g')
            .attr('class', 'dns-zone')
            .style('opacity', 0);

        // Add zone hull path with hierarchy-aware styling
        newZones.append('path')
            .attr('class', 'zone-hull')
            .attr('fill', d => {
                console.log(`Setting zone ${d.name} color to ${d.color}`);
                return d.color;
            })
            .attr('stroke', d => d3.color(d.color).darker(0.5))
            .attr('stroke-width', d => {
                // Broader zones get thicker strokes
                const depth = d.name.split('.').length;
                return Math.max(1, 5 - depth);
            })
            .attr('stroke-dasharray', d => {
                // More specific zones get dashed lines
                const depth = d.name.split('.').length;
                return depth > 4 ? '5,5' : 'none';
            })
            .style('fill-opacity', d => {
                // Broader zones get lower opacity so inner zones are more visible
                const depth = d.name.split('.').length;
                return Math.max(0.1, 0.35 - (depth * 0.03));
            })
            .style('stroke-opacity', 0.8)
            .style('cursor', 'pointer')
            .on('click', (event, d) => {
                event.stopPropagation();
                this.selectDNSZone(d);
            });

        // Add zone label
        newZones.append('text')
            .attr('class', 'zone-label')
            .attr('text-anchor', 'middle')
            .attr('font-weight', 'bold')
            .attr('font-size', d => {
                // Broader zones get larger text
                const depth = d.name.split('.').length;
                return Math.max(12, 20 - depth) + 'px';
            })
            .attr('fill', d => d3.color(d.color).darker(2))
            .style('cursor', 'pointer')
            .text(d => {
                console.log(`Adding label for zone: ${d.name}`);
                return d.name;
            })
            .on('click', (event, d) => {
                event.stopPropagation();
                this.selectDNSZone(d);
            });

        // Update all zones
        const allZones = newZones.merge(zones);
        allZones.transition()
            .duration(300)
            .style('opacity', 1);
            
        console.log(`DNS zones rendered: ${allZones.size()} elements`);
    }

    updateDNSZones(g) {
        // Update DNS zone hulls based on current node positions
        g.selectAll('.dns-zone').each((zoneData, i, nodes) => {
            const zoneElement = d3.select(nodes[i]);
            
            // Get nodes belonging to this zone
            const zoneNodes = this.nodes.filter(node => 
                zoneData.nodes.includes(node.id) && 
                node.x !== undefined && 
                node.y !== undefined &&
                !node.hidden
            );

            console.log(`Zone ${zoneData.name}: found ${zoneNodes.length} visible nodes out of ${zoneData.nodes.length} total nodes`);

            if (zoneNodes.length < 1) {
                // Hide zones with no visible nodes
                zoneElement.style('opacity', 0);
                return;
            }

            zoneElement.style('opacity', 1);

            // Calculate hull points
            const points = zoneNodes.map(node => [node.x, node.y]);
            
            if (zoneNodes.length === 1) {
                // For single nodes, create a circle around the node
                const [x, y] = points[0];
                // Broader zones get larger radius
                const depth = zoneData.name.split('.').length;
                const radius = Math.max(30, 70 - (depth * 8));
                const circlePoints = [];
                for (let angle = 0; angle < 2 * Math.PI; angle += Math.PI / 6) {
                    circlePoints.push([
                        x + Math.cos(angle) * radius,
                        y + Math.sin(angle) * radius
                    ]);
                }

                // Create smooth circle path
                const line = d3.line()
                    .x(d => d[0])
                    .y(d => d[1])
                    .curve(d3.curveCatmullRomClosed.alpha(0.5));

                // Update hull path
                zoneElement.select('.zone-hull')
                    .attr('d', line(circlePoints));

                // Update label position (center of circle)
                zoneElement.select('.zone-label')
                    .attr('x', x)
                    .attr('y', y - 50); // Above the circle
                
                // Update traffic flow labels position
                zoneElement.selectAll('.traffic-flow-label')
                    .attr('x', x)
                    .attr('y', y - 50);

            } else {
                // Multiple nodes - use hull calculation
                
                // Add padding around nodes for better visual grouping
                // Broader zones get more padding for better visual hierarchy
                const depth = zoneData.name.split('.').length;
                const padding = Math.max(20, 60 - (depth * 8));
                const expandedPoints = [];
                
                points.forEach(point => {
                    const [x, y] = point;
                    // Add multiple points around each node for better hull calculation
                    for (let angle = 0; angle < 2 * Math.PI; angle += Math.PI / 3) {
                        expandedPoints.push([
                            x + Math.cos(angle) * padding,
                            y + Math.sin(angle) * padding
                        ]);
                    }
                });

                // Calculate convex hull
                const hull = d3.polygonHull(expandedPoints);
                
                if (hull && hull.length >= 3) {
                    // Create smooth path
                    const line = d3.line()
                        .x(d => d[0])
                        .y(d => d[1])
                        .curve(d3.curveCatmullRomClosed.alpha(0.5));

                    // Update hull path
                    zoneElement.select('.zone-hull')
                        .attr('d', line(hull));

                    // Update label position (center of hull)
                    const centroid = d3.polygonCentroid(hull);
                    zoneElement.select('.zone-label')
                        .attr('x', centroid[0])
                        .attr('y', centroid[1]);
                    
                    // Update traffic flow labels position
                    zoneElement.selectAll('.traffic-flow-label')
                        .attr('x', centroid[0])
                        .attr('y', centroid[1]);
                }
            }
        });
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
        this.svg.selectAll('.dns-zone').classed('selected', false);
        
        // Select current node
        this.svg.selectAll('.node')
            .filter(d => d.id === node.id)
            .classed('selected', true);

        this.selectedNode = node;
        this.selectedDNSZone = null;
        this.updateInfoPanel(node);
    }

    selectDNSZone(zone) {
        // Remove previous selection
        this.svg.selectAll('.node').classed('selected', false);
        this.svg.selectAll('.dns-zone').classed('selected', false);
        
        // Select current zone
        this.svg.selectAll('.dns-zone')
            .filter(d => d.name === zone.name)
            .classed('selected', true);

        this.selectedNode = null;
        this.selectedDNSZone = zone;
        this.updateInfoPanelForZone(zone);
    }

    clearSelection() {
        // Remove all selections
        this.svg.selectAll('.node').classed('selected', false);
        this.svg.selectAll('.dns-zone').classed('selected', false);
        
        this.selectedNode = null;
        this.selectedDNSZone = null;
        
        // Reset info panel
        const infoContent = document.getElementById('info-content');
        infoContent.innerHTML = '<p>Click on a node or DNS zone to see details</p>';
    }

    updateInfoPanel(node) {
        const infoContent = document.getElementById('info-content');
        
        if (!node) {
            infoContent.innerHTML = '<p>Click on a node to see resource details</p>';
            return;
        }

        // Show basic info immediately
        this.showBasicNodeInfo(node);
        
        // Skip detailed loading for Listener nodes (they don't have full K8s resources)
        if (node.type === 'Listener') {
            return;
        }
        
        // Load detailed resource information
        this.loadResourceDetails(node);
    }

    updateInfoPanelForZone(zone) {
        const infoContent = document.getElementById('info-content');
        
        // Get nodes belonging to this zone
        const zoneNodes = this.nodes.filter(node => 
            zone.nodes.includes(node.id)
        );

        // Group nodes by type
        const nodesByType = zoneNodes.reduce((acc, node) => {
            if (!acc[node.type]) {
                acc[node.type] = [];
            }
            acc[node.type].push(node);
            return acc;
        }, {});

        let html = `
            <h4>🌐 DNS Zone</h4>
            <div class="resource-metadata">
                <span class="label">Zone Name:</span>
                <span class="value">${zone.name}</span>
                <span class="label">Total Resources:</span>
                <span class="value">${zoneNodes.length}</span>
            </div>
        `;

        // Add zone description
        html += `
            <div class="resource-section">
                <h5>Zone Information</h5>
                <div class="resource-section-content">
                    <div style="padding: 0.5rem; background: #f8f9fa; border-radius: 4px; margin-bottom: 1rem;">
                        <div style="font-size: 0.9rem; color: #495057;">
                            This DNS zone groups resources that handle traffic for the <strong>${zone.name}</strong> domain.
                            Resources in this zone are visually grouped together with a colored boundary.
                        </div>
                    </div>
                </div>
            </div>
        `;

        // Show resources by type
        Object.keys(nodesByType).sort().forEach(type => {
            const nodes = nodesByType[type];
            html += `
                <div class="resource-section">
                    <h5>${type} Resources (${nodes.length})</h5>
                    <div class="resource-section-content">
                        ${nodes.map(node => `
                            <div style="margin-bottom: 0.5rem; padding: 0.5rem; background: #f8f9fa; border-radius: 4px; cursor: pointer;" 
                                 onclick="visualizer.selectNode(visualizer.nodes.find(n => n.id === '${node.id}'))">
                                <strong>${node.name}</strong>
                                ${node.namespace ? ` (${node.namespace})` : ' (cluster-scoped)'}
                                <div style="font-size: 0.85rem; color: #6c757d;">
                                    ${node.hostname ? `Hostname: ${node.hostname}` : ''}
                                    ${node.listenerData ? `Port: ${node.listenerData.port} (${node.listenerData.protocol})` : ''}
                                </div>
                            </div>
                        `).join('')}
                    </div>
                </div>
            `;
        });

        // Add zone hierarchy information
        const depth = zone.name.split('.').length;
        if (depth > 2) {
            html += `
                <div class="resource-section">
                    <h5>Zone Hierarchy</h5>
                    <div class="resource-section-content">
                        <div style="padding: 0.5rem; background: #f8f9fa; border-radius: 4px;">
                            <div style="font-size: 0.9rem; color: #495057;">
                                This zone is <strong>level ${depth}</strong> in the DNS hierarchy.
                                ${depth > 4 ? 'More specific zones are rendered with dashed borders.' : 'Broader zones are rendered behind more specific ones.'}
                            </div>
                        </div>
                    </div>
                </div>
            `;
        }

        infoContent.innerHTML = html;
    }

    showBasicNodeInfo(node) {
        const infoContent = document.getElementById('info-content');
        
        let html = `
            <h4>${node.type}</h4>
            <div class="resource-metadata">
                <span class="label">Name:</span>
                <span class="value">${node.name}</span>
                <span class="label">Namespace:</span>
                <span class="value">${node.namespace || 'cluster-scoped'}</span>
                <span class="label">Kind:</span>
                <span class="value">${node.kind}</span>
                <span class="label">Group:</span>
                <span class="value">${node.group}</span>
                <span class="label">Version:</span>
                <span class="value">${node.version}</span>
            </div>
        `;

        // Add listener-specific information
        if (node.type === 'Listener' && node.listenerData) {
            html += `
                <div class="resource-section">
                    <h5>Listener Configuration</h5>
                    <div class="resource-section-content">
                        <div class="resource-metadata">
                            <span class="label">Port:</span>
                            <span class="value">${node.listenerData.port}</span>
                            <span class="label">Protocol:</span>
                            <span class="value">${node.listenerData.protocol}</span>
                            ${node.listenerData.hostname ? `
                                <span class="label">Hostname:</span>
                                <span class="value">${node.listenerData.hostname}</span>
                            ` : ''}
                            <span class="label">TLS:</span>
                            <span class="value">${node.listenerData.tls ? 'Yes' : 'No'}</span>
                        </div>
                    </div>
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
                    <div class="resource-section">
                        <h5>Listeners (${listenerNodes.length})</h5>
                        <div class="resource-section-content">
                            ${listenerNodes.map(listener => `
                                <div style="margin-bottom: 0.5rem;">
                                    <strong>Port ${listener.listenerData.port}</strong> (${listener.listenerData.protocol})
                                    ${listener.listenerData.hostname ? ` - ${listener.listenerData.hostname}` : ''}
                                </div>
                            `).join('')}
                        </div>
                    </div>
                `;
            }
        }

        // Add HTTPRoute-specific information showing related resources
        if (node.type === 'HTTPRoute') {
            // Find related DNSRecords (in the same DNS zone)
            const relatedDNSRecords = this.nodes
                .filter(n => n.type === 'DNSRecord' && node.dnsZone && n.dnsZone === node.dnsZone);

            // Find related Services (linked by backendRef)
            const relatedServices = this.links
                .filter(link => link.type === 'backendRef' && this.nodes[link.source]?.id === node.id)
                .map(link => this.nodes[link.target])
                .filter(n => n && n.type === 'Service');

            if (node.dnsZone) {
                html += `
                    <div class="resource-section">
                        <h5>🌐 DNS Zone</h5>
                        <div class="resource-section-content">
                            <div style="padding: 0.5rem; background: #f8f9fa; border-radius: 4px;">
                                <strong>${node.dnsZone}</strong>
                                <div style="font-size: 0.85rem; color: #6c757d;">This route belongs to the DNS zone shown as a colored area</div>
                            </div>
                        </div>
                    </div>
                `;
            }

            if (relatedDNSRecords.length > 0) {
                html += `
                    <div class="resource-section">
                        <h5>🌐 Related DNS Records (${relatedDNSRecords.length})</h5>
                        <div class="resource-section-content">
                            ${relatedDNSRecords.map(dns => `
                                <div style="margin-bottom: 0.5rem; padding: 0.5rem; background: #f8f9fa; border-radius: 4px;">
                                    <strong>${dns.name}</strong> (${dns.namespace || 'cluster-scoped'})
                                    <div style="font-size: 0.85rem; color: #6c757d;">DNS record in the same zone</div>
                                </div>
                            `).join('')}
                        </div>
                    </div>
                `;
            }

            if (relatedServices.length > 0) {
                html += `
                    <div class="resource-section">
                        <h5>🎯 Backend Services (${relatedServices.length})</h5>
                        <div class="resource-section-content">
                            ${relatedServices.map(service => `
                                <div style="margin-bottom: 0.5rem; padding: 0.5rem; background: #f8f9fa; border-radius: 4px;">
                                    <strong>${service.name}</strong> (${service.namespace || 'cluster-scoped'})
                                    <div style="font-size: 0.85rem; color: #6c757d;">Traffic is routed to this service</div>
                                </div>
                            `).join('')}
                        </div>
                    </div>
                `;
            }
        }

        // Add DNSRecord-specific information showing traffic flow
        if (node.type === 'DNSRecord') {
            // Find related HTTPRoutes (in the same DNS zone)
            const relatedHTTPRoutes = this.nodes
                .filter(n => n.type === 'HTTPRoute' && node.dnsZone && n.dnsZone === node.dnsZone);

            if (node.dnsZone) {
                html += `
                    <div class="resource-section">
                        <h5>🌐 DNS Zone</h5>
                        <div class="resource-section-content">
                            <div style="padding: 0.5rem; background: #f8f9fa; border-radius: 4px;">
                                <strong>${node.dnsZone}</strong>
                                <div style="font-size: 0.85rem; color: #6c757d;">This DNS record belongs to the zone shown as a colored area</div>
                            </div>
                        </div>
                    </div>
                `;
            }

            if (relatedHTTPRoutes.length > 0) {
                html += `
                    <div class="resource-section">
                        <h5>🔗 Related HTTP Routes (${relatedHTTPRoutes.length})</h5>
                        <div class="resource-section-content">
                            ${relatedHTTPRoutes.map(route => `
                                <div style="margin-bottom: 0.5rem; padding: 0.5rem; background: #f8f9fa; border-radius: 4px;">
                                    <strong>${route.name}</strong> (${route.namespace || 'cluster-scoped'})
                                    <div style="font-size: 0.85rem; color: #6c757d;">Route in the same DNS zone</div>
                                </div>
                            `).join('')}
                        </div>
                    </div>
                `;
            }
        }

        infoContent.innerHTML = html;
    }

    async loadResourceDetails(node) {
        const infoContent = document.getElementById('info-content');
        
        // Show loading indicator
        const loadingHtml = infoContent.innerHTML + `
            <div class="resource-section">
                <h5><span class="loading-spinner"></span>Loading detailed information...</h5>
            </div>
        `;
        infoContent.innerHTML = loadingHtml;

        try {
            const resourceType = node.type.toLowerCase();
            const url = `/api/resource/${resourceType}/${node.name}${node.namespace ? `?namespace=${node.namespace}` : ''}`;
            
            const response = await fetch(url);
            if (!response.ok) {
                throw new Error(`Failed to load resource details: ${response.status}`);
            }
            
            const resourceData = await response.json();
            this.showDetailedResourceInfo(node, resourceData);
            
        } catch (error) {
            console.error('Error loading resource details:', error);
            this.showResourceError(node, error.message);
        }
    }

    showDetailedResourceInfo(node, resourceData) {
        const infoContent = document.getElementById('info-content');
        
        let html = `
            <h4>${node.type}</h4>
            <div class="resource-metadata">
                <span class="label">Name:</span>
                <span class="value">${node.name}</span>
                <span class="label">Namespace:</span>
                <span class="value">${node.namespace || 'cluster-scoped'}</span>
                <span class="label">Kind:</span>
                <span class="value">${node.kind}</span>
                <span class="label">Group:</span>
                <span class="value">${node.group}</span>
                <span class="label">Version:</span>
                <span class="value">${node.version}</span>
            </div>
        `;

        // Add metadata section
        if (resourceData.metadata) {
            html += `
                <div class="resource-section">
                    <h5>Metadata</h5>
                    <div class="resource-section-content">
                        <div class="resource-metadata">
                            ${resourceData.metadata.uid ? `
                                <span class="label">UID:</span>
                                <span class="value">${resourceData.metadata.uid}</span>
                            ` : ''}
                            ${resourceData.metadata.creationTimestamp ? `
                                <span class="label">Created:</span>
                                <span class="value">${new Date(resourceData.metadata.creationTimestamp).toLocaleString()}</span>
                            ` : ''}
                            ${resourceData.metadata.resourceVersion ? `
                                <span class="label">Resource Version:</span>
                                <span class="value">${resourceData.metadata.resourceVersion}</span>
                            ` : ''}
                        </div>
                        ${resourceData.metadata.labels ? `
                            <div style="margin-top: 1rem;">
                                <strong>Labels:</strong>
                                <div style="margin-top: 0.5rem; font-family: monospace; font-size: 0.8rem;">
                                    ${Object.entries(resourceData.metadata.labels).map(([key, value]) => 
                                        `<div>${key}: ${value}</div>`
                                    ).join('')}
                                </div>
                            </div>
                        ` : ''}
                        ${resourceData.metadata.annotations ? `
                            <div style="margin-top: 1rem;">
                                <strong>Annotations:</strong>
                                <div style="margin-top: 0.5rem; font-family: monospace; font-size: 0.8rem;">
                                    ${Object.entries(resourceData.metadata.annotations).map(([key, value]) => 
                                        `<div>${key}: ${value}</div>`
                                    ).join('')}
                                </div>
                            </div>
                        ` : ''}
                    </div>
                </div>
            `;
        }

        // Add status section
        if (resourceData.status) {
            html += `
                <div class="resource-section">
                    <h5>Status</h5>
                    <div class="resource-section-content">
                        ${this.formatStatus(resourceData.status)}
                    </div>
                </div>
            `;
        }

        // Add spec section
        if (resourceData.spec) {
            html += `
                <div class="resource-section">
                    <h5>Specification</h5>
                    <div class="resource-section-content">
                        ${this.formatSpec(resourceData.spec, node.type)}
                    </div>
                </div>
            `;
        }

        // Add edit controls
        html += `
            <div class="edit-controls">
                <button class="btn-primary" onclick="window.gatewayGraph.startEditing('${node.type}', '${node.name}', '${node.namespace || ''}')">
                    Edit Resource
                </button>
                <button class="btn-secondary" onclick="window.gatewayGraph.viewFullYaml('${node.type}', '${node.name}', '${node.namespace || ''}')">
                    View Full YAML
                </button>
            </div>
        `;

        infoContent.innerHTML = html;
    }

    formatStatus(status) {
        let html = '';
        
        // Handle different types of status
        if (status.conditions) {
            html += '<div><strong>Conditions:</strong></div>';
            status.conditions.forEach(condition => {
                const statusClass = condition.status === 'True' ? 'status-ready' : 
                                  condition.status === 'False' ? 'status-error' : 'status-unknown';
                html += `
                    <div style="margin: 0.5rem 0; padding: 0.5rem; background: #f8f9fa; border-radius: 4px;">
                        <div><span class="status-indicator ${statusClass}"></span><strong>${condition.type}</strong></div>
                        <div style="font-size: 0.85rem; margin-top: 0.25rem;">Status: ${condition.status}</div>
                        ${condition.reason ? `<div style="font-size: 0.85rem;">Reason: ${condition.reason}</div>` : ''}
                        ${condition.message ? `<div style="font-size: 0.85rem;">Message: ${condition.message}</div>` : ''}
                    </div>
                `;
            });
        }
        
        // Add other status fields
        Object.entries(status).forEach(([key, value]) => {
            if (key !== 'conditions' && value !== null && value !== undefined) {
                html += `<div style="margin: 0.25rem 0;"><strong>${key}:</strong> ${JSON.stringify(value)}</div>`;
            }
        });
        
        return html || '<div>No status information available</div>';
    }

    formatSpec(spec, resourceType) {
        // Format spec based on resource type for better readability
        let html = '<pre style="background: #f8f9fa; padding: 0.75rem; border-radius: 4px; font-size: 0.8rem; overflow-x: auto;">';
        html += JSON.stringify(spec, null, 2);
        html += '</pre>';
        return html;
    }

    showResourceError(node, errorMessage) {
        const infoContent = document.getElementById('info-content');
        
        let html = `
            <h4>${node.type}</h4>
            <div class="resource-metadata">
                <span class="label">Name:</span>
                <span class="value">${node.name}</span>
                <span class="label">Namespace:</span>
                <span class="value">${node.namespace || 'cluster-scoped'}</span>
            </div>
            <div class="error-message">
                Failed to load detailed resource information: ${errorMessage}
            </div>
        `;
        
        infoContent.innerHTML = html;
    }

    async startEditing(resourceType, resourceName, namespace) {
        const infoContent = document.getElementById('info-content');
        
        // Show loading
        infoContent.innerHTML = `
            <div class="resource-section">
                <h5><span class="loading-spinner"></span>Loading resource for editing...</h5>
            </div>
        `;

        try {
            const url = `/api/resource/${resourceType.toLowerCase()}/${resourceName}${namespace ? `?namespace=${namespace}` : ''}`;
            const response = await fetch(url);
            
            if (!response.ok) {
                throw new Error(`Failed to load resource: ${response.status}`);
            }
            
            const resourceData = await response.json();
            this.showEditingInterface(resourceType, resourceName, namespace, resourceData);
            
        } catch (error) {
            console.error('Error loading resource for editing:', error);
            infoContent.innerHTML = `
                <div class="error-message">
                    Failed to load resource for editing: ${error.message}
                </div>
            `;
        }
    }

    showEditingInterface(resourceType, resourceName, namespace, resourceData) {
        const infoContent = document.getElementById('info-content');
        
        // Create a clean, editable version (like 'oc edit' does)
        const editableResource = this.createEditableResource(resourceData);
        const yamlContent = this.resourceToYaml(editableResource);
        
        const html = `
            <h4>Edit ${resourceType}: ${resourceName}</h4>
            <div class="resource-section">
                <h5>⚠️ Important Notes</h5>
                <div class="resource-section-content">
                    <div style="background: #fff3cd; border: 1px solid #ffeaa7; border-radius: 4px; padding: 0.75rem; margin-bottom: 1rem; font-size: 0.9rem;">
                        <strong>Immutable Fields:</strong> You cannot change:
                        <ul style="margin: 0.5rem 0 0 1.5rem;">
                            <li>Resource name (metadata.name)</li>
                            <li>Namespace (metadata.namespace)</li>
                        </ul>
                        <strong>What you can change:</strong> spec, labels, annotations
                        ${namespace && (namespace.includes('openshift') || namespace.includes('system')) ? `
                            <br/><br/>
                            <strong style="color: #d63384;">⚠️ Warning:</strong> This resource is in a system namespace (${namespace}). 
                            Changes may be reverted by system operators.
                        ` : ''}
                    </div>
                </div>
            </div>
            <div class="resource-section">
                <h5>Resource YAML (Editable)</h5>
                <div class="resource-section-content">
                    <textarea class="yaml-editor" id="yaml-editor">${yamlContent}</textarea>
                </div>
            </div>
            <div class="edit-controls">
                <button class="btn-success" onclick="window.gatewayGraph.saveResource('${resourceType}', '${resourceName}', '${namespace}')">
                    <span id="save-spinner" style="display: none;" class="loading-spinner"></span>
                    Save Changes
                </button>
                <button class="btn-secondary" onclick="window.gatewayGraph.cancelEditing('${resourceType}', '${resourceName}', '${namespace}')">
                    Cancel
                </button>
                <button class="btn-secondary" onclick="window.gatewayGraph.viewFullYaml('${resourceType}', '${resourceName}', '${namespace}')">
                    View Full YAML
                </button>
            </div>
            <div id="edit-messages"></div>
        `;
        
        infoContent.innerHTML = html;
    }

    async saveResource(resourceType, resourceName, namespace) {
        const yamlEditor = document.getElementById('yaml-editor');
        const saveSpinner = document.getElementById('save-spinner');
        const messagesDiv = document.getElementById('edit-messages');
        
        // Show loading
        saveSpinner.style.display = 'inline-block';
        messagesDiv.innerHTML = '';
        
        try {
            // Parse YAML back to JSON
            const resourceData = this.yamlToResource(yamlEditor.value);
            
            const url = `/api/resource/${resourceType.toLowerCase()}/${resourceName}${namespace ? `?namespace=${namespace}` : ''}`;
            const response = await fetch(url, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(resourceData)
            });
            
            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.error || `Failed to update resource: ${response.status}`);
            }
            
            // Show success message
            messagesDiv.innerHTML = '<div class="success-message">Resource updated successfully!</div>';
            
            // Refresh the graph to show updated data
            setTimeout(() => {
                this.loadData();
                // Go back to view mode
                this.cancelEditing(resourceType, resourceName, namespace);
            }, 1500);
            
        } catch (error) {
            console.error('Error saving resource:', error);
            messagesDiv.innerHTML = `<div class="error-message">Failed to save resource: ${error.message}</div>`;
        } finally {
            saveSpinner.style.display = 'none';
        }
    }

    cancelEditing(resourceType, resourceName, namespace) {
        // Find the node and reload its details
        const node = this.nodes.find(n => 
            n.type === resourceType && 
            n.name === resourceName && 
            (n.namespace || '') === (namespace || '')
        );
        
        if (node) {
            this.updateInfoPanel(node);
        }
    }

    createEditableResource(resource) {
        // Create a clean resource for editing, similar to what 'oc edit' shows
        // Remove read-only fields and system-generated metadata
        
        const editable = {
            apiVersion: resource.apiVersion || 'v1',
            kind: resource.kind || 'Unknown',
            metadata: {
                name: resource.metadata?.name || 'unknown'
            }
        };

        // Add namespace if present (skip for cluster-scoped resources)
        if (resource.metadata?.namespace) {
            editable.metadata.namespace = resource.metadata.namespace;
        }

        // Add editable metadata fields
        if (resource.metadata?.labels && Object.keys(resource.metadata.labels).length > 0) {
            editable.metadata.labels = { ...resource.metadata.labels };
        }

        if (resource.metadata?.annotations && Object.keys(resource.metadata.annotations).length > 0) {
            editable.metadata.annotations = { ...resource.metadata.annotations };
        }

        // Add spec if present
        if (resource.spec && Object.keys(resource.spec).length > 0) {
            editable.spec = JSON.parse(JSON.stringify(resource.spec)); // Deep copy
        }

        return editable;
    }

    async viewFullYaml(resourceType, resourceName, namespace) {
        const infoContent = document.getElementById('info-content');
        
        // Show loading
        infoContent.innerHTML = `
            <div class="resource-section">
                <h5><span class="loading-spinner"></span>Loading full resource YAML...</h5>
            </div>
        `;

        try {
            const url = `/api/resource/${resourceType.toLowerCase()}/${resourceName}${namespace ? `?namespace=${namespace}` : ''}`;
            const response = await fetch(url);
            
            if (!response.ok) {
                throw new Error(`Failed to load resource: ${response.status}`);
            }
            
            const resourceData = await response.json();
            const yamlContent = this.resourceToYaml(resourceData);
            
            const html = `
                <h4>${resourceType}: ${resourceName} (Full YAML)</h4>
                <div class="resource-section">
                    <h5>Complete Resource YAML (Read-Only)</h5>
                    <div class="resource-section-content">
                        <pre style="background: #f8f9fa; padding: 0.75rem; border-radius: 4px; font-size: 0.8rem; overflow-x: auto; white-space: pre-wrap;">${yamlContent}</pre>
                    </div>
                </div>
                <div class="edit-controls">
                    <button class="btn-primary" onclick="window.gatewayGraph.startEditing('${resourceType}', '${resourceName}', '${namespace}')">
                        Edit Resource
                    </button>
                    <button class="btn-secondary" onclick="window.gatewayGraph.cancelEditing('${resourceType}', '${resourceName}', '${namespace}')">
                        Back to Details
                    </button>
                </div>
            `;
            
            infoContent.innerHTML = html;
            
        } catch (error) {
            console.error('Error loading full YAML:', error);
            infoContent.innerHTML = `
                <div class="error-message">
                    Failed to load full resource YAML: ${error.message}
                </div>
            `;
        }
    }

    resourceToYaml(resource) {
        // Simple JSON to YAML converter for basic formatting
        return JSON.stringify(resource, null, 2);
    }

    yamlToResource(yamlString) {
        // Simple YAML to JSON converter (assumes JSON format)
        try {
            return JSON.parse(yamlString);
        } catch (error) {
            throw new Error('Invalid JSON/YAML format');
        }
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

    toggleDNSZones() {
        this.showDNSZones = !this.showDNSZones;
        
        // Update button text
        const button = document.getElementById('dns-zones-toggle-btn');
        button.textContent = `DNS Zones: ${this.showDNSZones ? 'ON' : 'OFF'}`;
        
        // Re-render the graph
        this.render();
    }
}

// Initialize the application when the DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    console.log('DOM loaded, initializing Gateway Graph Visualizer...');
    window.gatewayGraph = new GatewayGraphVisualizer();
    // Also create a global reference for onclick handlers
    window.visualizer = window.gatewayGraph;
}); 
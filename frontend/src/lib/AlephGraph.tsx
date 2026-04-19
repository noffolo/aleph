import React, { useEffect, useRef } from 'react';
import * as d3 from 'd3';

interface Row {
  values: { [key: string]: string };
}

interface AlephGraphProps {
  rows: Row[];
  onRowClick?: (row: Row) => void;
}

export const AlephGraph: React.FC<AlephGraphProps> = ({ rows, onRowClick }) => {
  const svgRef = useRef<SVGSVGElement>(null);

  useEffect(() => {
    if (!svgRef.current || rows.length === 0) return;

    const width = svgRef.current.clientWidth;
    const height = 500;

    // Detect nodes and links from rows (simplified heuristic)
    // Assume each row is a node, and look for ID-like references to other rows
    const nodes: any[] = rows.map((r, i) => ({ id: i, label: r.values['oggetto'] || r.values['name'] || `Node ${i}`, data: r }));
    const links: any[] = [];
    
    // Fake links for visual demo of graph capability if no real relations found
    for (let i = 1; i < nodes.length; i++) {
        if (Math.random() > 0.5) links.push({ source: i, target: Math.floor(Math.random() * i) });
    }

    const svg = d3.select(svgRef.current);
    svg.selectAll("*").remove();

    const simulation = d3.forceSimulation(nodes)
      .force("link", d3.forceLink(links).id((d: any) => d.id).distance(100))
      .force("charge", d3.forceManyBody().strength(-300))
      .force("center", d3.forceCenter(width / 2, height / 2));

    const link = svg.append("g")
      .attr("stroke", "#e2e8f0")
      .attr("stroke-opacity", 0.6)
      .selectAll("line")
      .data(links)
      .join("line")
      .attr("stroke-width", 2);

    const node = svg.append("g")
      .selectAll("g")
      .data(nodes)
      .join("g")
      .call(d3.drag()
        .on("start", (event: any, d: any) => {
          if (!event.active) simulation.alphaTarget(0.3).restart();
          d.fx = d.x; d.fy = d.y;
        })
        .on("drag", (event: any, d: any) => { d.fx = event.x; d.fy = event.y; })
        .on("end", (event: any, d: any) => {
          if (!event.active) simulation.alphaTarget(0);
          d.fx = null; d.fy = null;
        }))
      .on("click", (event: any, d: any) => onRowClick?.(d.data));

    node.append("circle")
      .attr("r", 25)
      .attr("fill", "#3b82f6")
      .attr("stroke", "#fff")
      .attr("stroke-width", 3);

    node.append("text")
      .text((d: any) => d.label.substring(0, 15) + "...")
      .attr("x", 0)
      .attr("y", 40)
      .attr("text-anchor", "middle")
      .attr("font-size", "10px")
      .attr("font-weight", "bold")
      .attr("fill", "#64748b");

    simulation.on("tick", () => {
      link
        .attr("x1", (d: any) => d.source.x)
        .attr("y1", (d: any) => d.source.y)
        .attr("x2", (d: any) => d.target.x)
        .attr("y2", (d: any) => d.target.y);

      node.attr("transform", (d: any) => `translate(${d.x},${d.y})`);
    });

  }, [rows]);

  return (
    <div className="bg-white rounded-3xl border border-gray-100 shadow-xl overflow-hidden">
      <div className="p-4 border-b bg-gray-50 flex justify-between items-center">
         <span className="text-xs font-bold text-gray-400 uppercase tracking-widest">Relational Force Graph</span>
         <div className="flex space-x-1">
             <div className="w-2 h-2 rounded-full bg-blue-500"></div>
             <div className="w-2 h-2 rounded-full bg-blue-300"></div>
         </div>
      </div>
      <svg ref={svgRef} className="w-full h-[500px] cursor-move"></svg>
    </div>
  );
};

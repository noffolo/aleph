import React, { useEffect, useRef } from 'react';

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
    const loadD3 = async () => {
      const d3 = await import('d3');
      
      if (!svgRef.current || rows.length === 0) return;

      const width = svgRef.current.clientWidth;
      const height = 500;

      const nodes: any[] = rows.map((r, i) => ({ id: i, label: r.values['oggetto'] || r.values['name'] || r.values['title'] || `Node ${i}`, data: r }));
      const links: any[] = [];

      const idMap = new Map<string, number>();
      const labelMap = new Map<string, number>();
      nodes.forEach((n, i) => {
        const rowVals = rows[i].values;
        const rowId = rowVals['id'] || rowVals[n.label.toLowerCase() + '_id'] || '';
        if (rowId) idMap.set(rowId, i);
        labelMap.set(n.label.toLowerCase(), i);
      });

      rows.forEach((r, i) => {
        const vals = r.values;
        Object.entries(vals).forEach(([key, val]) => {
          if (!val) return;
          const lk = key.toLowerCase();
          if (lk.endsWith('_id') || lk === 'parent' || lk === 'related' || lk === 'relation') {
            const refIdx = idMap.get(val) ?? labelMap.get(val.toLowerCase());
            if (refIdx !== undefined && refIdx !== i) {
              links.push({ source: i, target: refIdx });
            }
          }
        });
      });

      const svg = d3.select(svgRef.current);
      svg.selectAll("*").remove();

      const simulation = d3.forceSimulation(nodes)
        .force("link", d3.forceLink(links).id((d: any) => d.id).distance(100))
        .force("charge", d3.forceManyBody().strength(-300))
        .force("center", d3.forceCenter(width / 2, height / 2));

      const link = svg.append("g")
        .attr("stroke", "#2a2a3a")
        .attr("stroke-opacity", 0.6)
        .selectAll("line")
        .data(links)
        .join("line")
        .attr("stroke-width", 1);

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
        .attr("r", 20)
        .attr("fill", "#00d4ff")
        .attr("fill-opacity", 0.15)
        .attr("stroke", "#00d4ff")
        .attr("stroke-width", 1.5);

      node.append("text")
        .text((d: any) => d.label.length > 15 ? d.label.substring(0, 15) + "…" : d.label)
        .attr("x", 0)
        .attr("y", 32)
        .attr("text-anchor", "middle")
        .attr("font-size", "10px")
        .attr("font-family", "monospace")
        .attr("fill", "#e4e4e7");

      simulation.on("tick", () => {
        link
          .attr("x1", (d: any) => d.source.x)
          .attr("y1", (d: any) => d.source.y)
          .attr("x2", (d: any) => d.target.x)
          .attr("y2", (d: any) => d.target.y);

        node.attr("transform", (d: any) => `translate(${d.x},${d.y})`);
      });

      return simulation;
    };

    const simulation = loadD3();

    return () => {
      simulation.then(sim => sim?.stop());
    };
  }, [rows]);


  return (
    <div className="bg-surface border border-border overflow-hidden">
      <div className="h-9 flex items-center justify-between px-4 border-b border-border shrink-0">
         <span className="text-[10px] font-mono font-bold text-textDim uppercase tracking-widest">Relational Force Graph</span>
         <div className="flex space-x-1">
             <div className="w-1.5 h-1.5 rounded-full bg-primary"></div>
             <div className="w-1.5 h-1.5 rounded-full bg-textDim"></div>
         </div>
      </div>
      <svg ref={svgRef} className="w-full h-[500px] cursor-move"></svg>
    </div>
  );
};
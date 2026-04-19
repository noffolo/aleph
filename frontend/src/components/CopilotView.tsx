import React from 'react';
import { Bot, Send } from 'lucide-react';

interface ChatMessage {
  role: string;
  content: string;
  toolCall?: string;
}

interface Agent {
  id: string;
  name: string;
  model: string;
}

interface CopilotViewProps {
  agents: Agent[];
  selectedAgent: string;
  setSelectedAgent: (id: string) => void;
  chat: ChatMessage[];
  input: string;
  setInput: (val: string) => void;
  onSend: () => void;
  isStreaming: boolean;
}

export const CopilotView: React.FC<CopilotViewProps> = ({
  agents, selectedAgent, setSelectedAgent,
  chat, input, setInput, onSend, isStreaming
}) => {
  return (
    <div className="max-w-4xl mx-auto h-full flex flex-col">
       <div className="mb-6 flex items-center space-x-3 bg-white p-3 rounded-2xl border border-gray-100 shadow-sm">
          <Bot size={20} className="text-blue-600" />
          <select 
            value={selectedAgent} 
            onChange={(e) => setSelectedAgent(e.target.value)} 
            className="flex-1 bg-transparent outline-none text-sm font-bold"
          >
            <option value="">Scegli un collaboratore AI...</option>
            {agents.map(a => <option key={a.id} value={a.id}>{a.name} ({a.model})</option>)}
          </select>
       </div>

       <div className="flex-1 space-y-4 mb-6 overflow-auto pr-4 custom-scrollbar">
          {chat.map((msg, i) => (
            <div key={i} className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}>
              <div className="flex flex-col space-y-1.5 max-w-[85%]">
                {msg.role === 'assistant' && <div className="text-[9px] font-bold text-gray-400 uppercase tracking-widest ml-1">Copilot</div>}
                <div className={`p-5 rounded-2xl ${msg.role === 'user' ? 'bg-blue-600 text-white shadow-lg shadow-blue-100' : 'bg-white border border-gray-100 text-gray-800 shadow-sm'}`}>
                  <p className="text-[15px] leading-relaxed whitespace-pre-wrap">{msg.content}</p>
                </div>
                {msg.toolCall && (
                  <div className="text-[10px] font-mono bg-amber-50 text-amber-700 p-2.5 rounded-xl border border-amber-100 flex items-center space-x-2">
                     <span className="w-1.5 h-1.5 bg-amber-400 rounded-full animate-pulse"></span>
                     <span>{msg.toolCall}</span>
                  </div>
                )}
              </div>
            </div>
          ))}
          {isStreaming && (
            <div className="flex items-center space-x-2 text-gray-400 text-xs animate-pulse ml-1">
               <div className="flex space-x-1">
                  <div className="w-1 h-1 bg-gray-300 rounded-full animate-bounce"></div>
                  <div className="w-1 h-1 bg-gray-300 rounded-full animate-bounce [animation-delay:0.2s]"></div>
                  <div className="w-1 h-1 bg-gray-300 rounded-full animate-bounce [animation-delay:0.4s]"></div>
               </div>
               <span>L'Agente sta elaborando la risposta...</span>
            </div>
          )}
       </div>

       <div className="relative pb-6">
         <input 
            value={input} 
            onChange={(e) => setInput(e.target.value)} 
            onKeyDown={(e) => e.key === 'Enter' && onSend()} 
            disabled={isStreaming} 
            className="w-full p-6 pr-16 bg-white border border-gray-200 rounded-3xl focus:outline-none focus:ring-4 focus:ring-blue-500/10 transition-all shadow-xl text-lg placeholder:text-gray-300" 
            placeholder="Assegna un compito o fai una domanda..." 
         />
         <button 
            onClick={onSend} 
            disabled={isStreaming || !input} 
            className="absolute right-3.5 top-3.5 p-3 bg-blue-600 text-white rounded-2xl hover:bg-blue-700 transition-all shadow-lg shadow-blue-200 disabled:opacity-50 disabled:grayscale"
         >
            <Send size={24} />
         </button>
       </div>
    </div>
  );
};

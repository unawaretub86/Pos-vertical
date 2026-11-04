"use client";

import { useCallback, useEffect, useState, useRef } from "react";
import { api } from "@/lib/api";
import { useBusinessConfig } from "@/lib/business-config-context";
import { useRouter } from "next/navigation";
import { FiVolume2, FiVolumeX, FiAlertCircle } from "react-icons/fi";
import { useIsMobile } from "@/hooks/useIsMobile";

type KDSItem = {
  id: number;
  product_name: string;
  quantity: number;
  note: string;
  status: string;
  created_at: string;
};

type KDSOrder = {
  order_id: number;
  type: string;
  table_name: string;
  turn_number?: number;
  created_at: string;
  items: KDSItem[];
};

type KDSResponse = {
  server_time: string;
  orders: KDSOrder[];
};

export default function BarraPage() {
  const router = useRouter();
  const { config, loading } = useBusinessConfig();
  const [data, setData] = useState<KDSResponse | null>(null);
  const [error, setError] = useState(false);
  const [soundEnabled, setSoundEnabled] = useState(false);
  const isMobile = useIsMobile();
  
  const knownItemIds = useRef<Set<number>>(new Set());
  const initialLoad = useRef(true);
  const audioContext = useRef<AudioContext | null>(null);

  useEffect(() => {
    if (!loading && (!config || !config.restaurant_mode)) {
      router.push("/");
    }
  }, [loading, config, router]);

  const playBeep = useCallback(() => {
    if (!soundEnabled) return;
    try {
      if (!audioContext.current) {
        audioContext.current = new (window.AudioContext || (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext)();
      }
      const osc = audioContext.current.createOscillator();
      const gainNode = audioContext.current.createGain();
      osc.type = "sine";
      osc.frequency.setValueAtTime(880, audioContext.current.currentTime);
      gainNode.gain.setValueAtTime(0.1, audioContext.current.currentTime);
      osc.connect(gainNode);
      gainNode.connect(audioContext.current.destination);
      osc.start();
      osc.stop(audioContext.current.currentTime + 0.3);
    } catch (e) {
      console.error(e);
    }
  }, [soundEnabled]);

  const loadData = useCallback(async () => {
    try {
      const res = await api.get<KDSResponse>("/kds/orders");
      setError(false);
      
      let hasNewItems = false;
      const currentIds = new Set<number>();
      
      res.orders.forEach(o => {
        o.items.forEach(i => {
          currentIds.add(i.id);
          if (!knownItemIds.current.has(i.id)) {
            hasNewItems = true;
          }
        });
      });
      
      knownItemIds.current = currentIds;
      
      if (!initialLoad.current && hasNewItems) {
        playBeep();
      }
      
      initialLoad.current = false;
      setData(res);
    } catch (err) {
      console.error("Error loading KDS", err);
      setError(true);
    }
  }, [playBeep]);

  // loadData only changes identity with soundEnabled, so the polling restarts exactly then.
  useEffect(() => {
    queueMicrotask(loadData);
    const interval = setInterval(loadData, 3000);
    return () => clearInterval(interval);
  }, [loadData]);

  const advanceItem = async (orderId: number, itemId: number, currentStatus: string) => {
    let nextStatus = "";
    if (currentStatus === "pending") nextStatus = "preparing";
    else if (currentStatus === "preparing") nextStatus = "ready";
    else return;

    try {
      await api.patch(`/orders/${orderId}/items/${itemId}`, { status: nextStatus });
      loadData();
    } catch (err) {
      console.error("Error advancing item", err);
    }
  };

  const getMinutes = (created_at: string, server_time: string) => {
    const created = new Date(created_at).getTime();
    const server = new Date(server_time).getTime();
    return Math.floor((server - created) / 60000);
  };

  const getColorClass = (mins: number) => {
    if (mins >= 10) return "bg-red-100 border-red-500 text-red-900";
    if (mins >= 5) return "bg-yellow-100 border-yellow-500 text-yellow-900";
    return "bg-white border-gray-200 text-gray-900";
  };

  if (loading) return null;

  const renderColumn = (title: string, status: string) => {
    const filteredOrders = data?.orders.map(o => ({
      ...o,
      items: o.items.filter(i => i.status === status)
    })).filter(o => o.items.length > 0) || [];

    return (
      <div className={"flex-1 flex flex-col gap-4 " + (isMobile ? "w-full" : "min-w-[300px]")}>
        <h2 className="text-xl font-bold p-3 bg-gray-100 text-center rounded-lg shadow-sm border border-gray-200">{title}</h2>
        {filteredOrders.length === 0 ? (
          <p className="text-gray-400 text-center py-8">Ningún ítem</p>
        ) : (
          filteredOrders.map(order => (
            <div key={order.order_id} className="flex flex-col gap-3">
              {order.items.map(item => {
                const mins = getMinutes(item.created_at, data!.server_time);
                return (
                  <div 
                    key={item.id} 
                    onClick={() => advanceItem(order.order_id, item.id, item.status)}
                    className={"p-4 border-l-4 rounded-lg shadow-sm cursor-pointer select-none active:opacity-75 transition-colors " + getColorClass(mins)}
                  >
                    <div className="flex justify-between items-start mb-2">
                      <span className="font-bold text-lg">{order.table_name || 'Sin Mesa'}</span>
                      <span className="font-mono text-sm font-semibold">{mins} min</span>
                    </div>
                    <div className="text-lg font-semibold">{item.quantity} x {item.product_name}</div>
                    {item.note && (
                      <div className="mt-2 bg-yellow-200 text-yellow-900 px-2 py-1 rounded font-bold text-sm inline-block">
                        {item.note}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          ))
        )}
      </div>
    );
  };

  const hasAnyItems = data && data.orders.some(o => o.items.length > 0);

  return (
    <div className="min-h-screen bg-gray-50 flex flex-col font-sans">
      <header className="bg-white border-b border-gray-200 p-4 flex justify-between items-center sticky top-0 z-10 shadow-sm">
        <h1 className="text-2xl font-black tracking-tight text-gray-800">Pantalla de Barra (KDS)</h1>
        <div className="flex items-center gap-4">
          {error && (
            <div className="flex items-center gap-2 text-red-600 bg-red-50 px-3 py-1 rounded-full border border-red-200 font-medium">
              <FiAlertCircle />
              <span>Sin conexión</span>
            </div>
          )}
          <button
            onClick={() => {
              if (!soundEnabled) {
                const ac = new (window.AudioContext || (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext)();
                audioContext.current = ac;
              }
              setSoundEnabled(!soundEnabled);
            }}
            className={"flex items-center gap-2 px-4 py-2 rounded-full font-bold transition-all " + (soundEnabled ? "bg-green-100 text-green-700 hover:bg-green-200" : "bg-gray-200 text-gray-700 hover:bg-gray-300")}
          >
            {soundEnabled ? <FiVolume2 className="text-xl" /> : <FiVolumeX className="text-xl" />}
            {soundEnabled ? "Sonido Activado" : "Activar Sonido"}
          </button>
        </div>
      </header>
      
      <main className="flex-1 p-4 overflow-auto">
        {!data ? (
          <div className="flex justify-center items-center h-full">
            <span className="text-gray-400 font-medium text-lg animate-pulse">Cargando...</span>
          </div>
        ) : !hasAnyItems ? (
          <div className="flex flex-col justify-center items-center h-full opacity-50 py-20">
            <h2 className="text-4xl font-black text-gray-400 mb-2 tracking-tight">Todo al día</h2>
            <p className="text-gray-400 text-lg">No hay pedidos pendientes en barra.</p>
          </div>
        ) : (
          <div className={"flex gap-6 " + (isMobile ? "flex-col" : "flex-row")}>
            {renderColumn("Pendiente", "pending")}
            {renderColumn("Preparando", "preparing")}
            {renderColumn("Listo", "ready")}
          </div>
        )}
      </main>
    </div>
  );
}

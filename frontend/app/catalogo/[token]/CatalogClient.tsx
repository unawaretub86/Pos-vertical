'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';

interface CatalogProduct {
  id: number;
  name: string;
  code: string;
  sale_price: number;
  unit: string;
  category: string;
}

interface CatalogData {
  business_name: string;
  logo: string;
  currency: string;
  products: CatalogProduct[];
}

export default function CatalogClient({ token }: { token: string }) {
  const [data, setData] = useState<CatalogData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');

  useEffect(() => {
    api.get<CatalogData>(`/public/catalog/${token}`)
      .then(setData)
      .catch(() => setError(true))
      .finally(() => setLoading(false));
  }, [token]);

  if (loading) {
    return <div style={{ padding: '20px', textAlign: 'center', fontFamily: 'system-ui' }}>Cargando catálogo...</div>;
  }

  if (error || !data) {
    return <div style={{ padding: '20px', textAlign: 'center', fontFamily: 'system-ui', color: '#666' }}>Catálogo no encontrado o no disponible.</div>;
  }

  const formatPrice = (price: number) => {
    return new Intl.NumberFormat('es-CO', { style: 'currency', currency: data.currency, minimumFractionDigits: 0 }).format(price);
  };

  const filteredProducts = data.products.filter(p => 
    p.name.toLowerCase().includes(searchTerm.toLowerCase()) || 
    (p.code && p.code.toLowerCase().includes(searchTerm.toLowerCase()))
  );

  return (
    <div style={{ fontFamily: 'system-ui, sans-serif', maxWidth: '800px', margin: '0 auto', padding: '16px' }}>
      <header style={{ display: 'flex', alignItems: 'center', gap: '16px', marginBottom: '24px', paddingBottom: '16px', borderBottom: '1px solid #eee' }}>
        {data.logo && (
          <img src={data.logo} alt="Logo" style={{ width: '60px', height: '60px', objectFit: 'contain', borderRadius: '8px' }} />
        )}
        <div>
          <h1 style={{ margin: 0, fontSize: '24px', color: '#111' }}>{data.business_name}</h1>
          <p style={{ margin: '4px 0 0 0', color: '#666', fontSize: '14px' }}>Catálogo de productos</p>
        </div>
      </header>

      <div style={{ marginBottom: '24px' }}>
        <input
          type="text"
          placeholder="Buscar producto..."
          value={searchTerm}
          onChange={e => setSearchTerm(e.target.value)}
          style={{ width: '100%', padding: '12px 16px', fontSize: '16px', borderRadius: '8px', border: '1px solid #ddd', boxSizing: 'border-box' }}
        />
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: '16px' }}>
        {filteredProducts.map(p => (
          <div key={p.id} style={{ border: '1px solid #eee', borderRadius: '8px', padding: '16px', display: 'flex', flexDirection: 'column' }}>
            <div style={{ fontSize: '12px', color: '#888', marginBottom: '4px', textTransform: 'uppercase', letterSpacing: '0.5px' }}>
              {p.category || 'Sin categoría'}
            </div>
            <div style={{ fontSize: '16px', fontWeight: 500, color: '#222', marginBottom: '8px', flex: 1 }}>
              {p.name}
            </div>
            {p.code && (
              <div style={{ fontSize: '12px', color: '#999', marginBottom: '12px' }}>
                Ref: {p.code}
              </div>
            )}
            <div style={{ fontSize: '18px', fontWeight: 700, color: '#000', marginTop: 'auto' }}>
              {formatPrice(p.sale_price)}
              <span style={{ fontSize: '12px', color: '#666', fontWeight: 400, marginLeft: '4px' }}>/ {p.unit}</span>
            </div>
          </div>
        ))}
        {filteredProducts.length === 0 && (
          <div style={{ gridColumn: '1 / -1', textAlign: 'center', padding: '40px', color: '#666' }}>
            No se encontraron productos.
          </div>
        )}
      </div>
      
      <footer style={{ marginTop: '40px', paddingTop: '20px', borderTop: '1px solid #eee', textAlign: 'center', color: '#aaa', fontSize: '12px' }}>
        Desarrollado por Pos-vertical
      </footer>
    </div>
  );
}

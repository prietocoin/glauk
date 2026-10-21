/**
 * @file componentLoader.js
 * @description Inyector atómico y recursivo de plantillas HTML parciales para glauk2.
 */

/**
 * Busca e inyecta dinámicamente los componentes HTML declarados con el atributo x-include.
 */
export async function cargarComponentes() {
  let nodos = Array.from(document.querySelectorAll('[x-include]'));

  while (nodos.length > 0) {
    await Promise.all(
      nodos.map(async (el) => {
        const url = el.getAttribute('x-include');
        el.removeAttribute('x-include');

        if (!url) return;

        try {
          const res = await fetch(url);
          if (res.ok) {
            el.innerHTML = await res.text();
          } else {
            console.error(`[componentLoader ❌] 404 No encontrado: ${url}`);
          }
        } catch (err) {
          console.error(`[componentLoader ❌] Error al cargar ${url}:`, err.message);
        }
      })
    );

    // Vuelve a consultar por si las plantillas recién inyectadas contienen subs-componentes
    nodos = Array.from(document.querySelectorAll('[x-include]'));
  }
}

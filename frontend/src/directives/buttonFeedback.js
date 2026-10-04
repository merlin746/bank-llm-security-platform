const bindings = new WeakMap()
const handledEvents = new WeakSet()

// Delegate feedback so dynamically rendered controls and navigation share it.
export const buttonFeedback = {
  mounted(root) {
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)')
    const active = new Map()

    function clear(button) {
      const feedback = active.get(button)
      if (!feedback) return
      active.delete(button)
      feedback.animation.cancel()
      feedback.node.remove()
    }

    function respond(event) {
      if (reducedMotion.matches || !(event.target instanceof Element)) return
      if (event.type === 'keydown' && (event.repeat || !['Enter', ' '].includes(event.key))) return
      if (event.type === 'pointerdown' && event.button !== 0) return
      const button = event.target.closest('button.el-button, button.preset-button, .platform-nav a')
      if (!button || !root.contains(button) || button.matches(':disabled, .is-disabled, .is-loading')) return
      if (event.type === 'keydown' && event.key === ' ' && button.matches('a')) return
      if (typeof button.animate !== 'function') return
      // A drawer can sit inside another feedback scope; acknowledge each event once.
      if (handledEvents.has(event)) return
      handledEvents.add(event)
      clear(button)

      const bounds = button.getBoundingClientRect()
      const diameter = Math.hypot(bounds.width, bounds.height) * 2
      const x = event.type === 'pointerdown' ? event.clientX - bounds.left : bounds.width / 2
      const y = event.type === 'pointerdown' ? event.clientY - bounds.top : bounds.height / 2
      const node = document.createElement('span')
      node.className = 'button-ripple'
      node.setAttribute('aria-hidden', 'true')
      Object.assign(node.style, {
        width: diameter + 'px', height: diameter + 'px',
        left: x - diameter / 2 + 'px', top: y - diameter / 2 + 'px'
      })
      button.appendChild(node)
      const animation = node.animate([
        { transform: 'scale(0)', opacity: 0.22 },
        { transform: 'scale(1)', opacity: 0 }
      ], { duration: 420, easing: 'cubic-bezier(0.16, 1, 0.3, 1)' })
      active.set(button, { node, animation })
      animation.onfinish = () => clear(button)
    }

    function stopMotion(event) {
      if (event.matches) for (const button of [...active.keys()]) clear(button)
    }
    root.addEventListener('pointerdown', respond)
    root.addEventListener('keydown', respond)
    reducedMotion.addEventListener('change', stopMotion)
    bindings.set(root, () => {
      root.removeEventListener('pointerdown', respond)
      root.removeEventListener('keydown', respond)
      reducedMotion.removeEventListener('change', stopMotion)
      for (const button of [...active.keys()]) clear(button)
    })
  },
  unmounted(root) {
    bindings.get(root)?.()
    bindings.delete(root)
  }
}

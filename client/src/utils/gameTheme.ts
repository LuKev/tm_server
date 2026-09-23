/** Canvas and DOM controls read the same gameplay theme values. */
export function gameThemeColor(element: Element, token: 'focus' | 'accent' | 'border-strong' | 'text'): string {
  const styles = getComputedStyle(element)
  return styles.getPropertyValue(`--game-${token}`).trim() || styles.color
}

export const getHoldingPnlValue = (holding) => {
  const quantity = holding?.quantity || 0
  const costPrice = holding?.costPrice || 0
  const currPrice = holding?.asset ? (holding.asset.currentPrice || 0) : 0
  return quantity * (currPrice - costPrice)
}

export const getHoldingPnlPct = (holding) => {
  const quantity = holding?.quantity || 0
  const costPrice = holding?.costPrice || 0
  const currPrice = holding?.asset ? (holding.asset.currentPrice || 0) : 0
  if (quantity === 0 || costPrice === 0) return 0

  const direction = quantity < 0 ? -1 : 1
  return ((currPrice - costPrice) / costPrice) * 100 * direction
}

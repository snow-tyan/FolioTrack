package controllers

import "errors"

func validateHoldingNumbers(market string, quantity, costPrice float64, allowClear bool) error {
	if allowClear && quantity == 0 && costPrice == 0 {
		return nil
	}

	if costPrice <= 0 {
		return errors.New("持仓成本价必须大于 0")
	}

	if market == "US-stock" {
		if quantity == 0 {
			return errors.New("美股持仓数量不能为 0，可为负数")
		}
		return nil
	}

	if quantity <= 0 {
		return errors.New("持仓数量必须大于 0")
	}

	return nil
}

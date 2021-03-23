package device

var chipId = map[uint32]string{
	0x410: "stm32f1xx medium-density",
	0x411: "stm32f2xx",
	0x412: "stm32f1xx low-density",
	0x413: "stm32f4xx",
	0x414: "stm32f1xx high-density",
	0x415: "stm32l4xx",
	0x416: "stm32l1xx medium-density",
	0x417: "stm32l0xx",
	0x418: "stm32f1xx connectivity line",
	0x419: "stm32f4xx high-density",
	0x420: "stm32f1xx value line low/medium-density",
	0x421: "stm32f446",
	0x422: "stm32f3xx",
	0x423: "stm32f4xx low power",
	0x425: "stm32l0xx cat. 2",
	0x427: "stm32l1xx medium-density/plus",
	0x428: "stm32f1xx value line high-density",
	0x429: "stm32l1xx cat. 2",
	0x430: "stm32f1xx xl-density",
	0x431: "stm32f411re",
	0x432: "stm32f37x",
	0x433: "stm32f4xx de",
	0x434: "stm32f4xx dsi",
	0x435: "stm32l43x",
	0x436: "stm32l1xx high-density",
	0x437: "stm32l152re",
	0x438: "stm32f334",
	0x439: "stm32f3xx small",
	0x440: "stm32f0xx",
	0x441: "stm32f412",
	0x442: "stm32f09x",
	0x444: "stm32f0xx small",
	0x445: "stm32f04x",
	0x446: "stm32f303 high-density",
	0x447: "stm32l0xx cat. 5",
	0x448: "stm32f0xx can",
	0x449: "stm32f7",
	0x451: "stm32f7xx",
	0x457: "stm32l011",
	0x458: "stm32f410",
	0x463: "stm32f413",
}

func IdentifyChip(chip_id uint32) string {
	cat := chipId[chip_id&0xFFF]
	if cat == "" {
		return "unknown"
	}
	return cat
}

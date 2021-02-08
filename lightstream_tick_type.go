package igmarkets

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type LightStreamChartTick struct {
	LTV              float64    `json:"LTV,omitempty"`              //Last traded volume
	TTV              float64    `json:"TTV,omitempty"`              //Incremental volume
	UTM              *time.Time `json:"UTM,omitempty"`              //Update time (as milliseconds from the Epoch)
	DAY_OPEN_MID     float64    `json:"DAY_OPEN_MID,omitempty"`     //Mid open price for the day
	DAY_NET_CHG_MID  float64    `json:"DAY_NET_CHG_MID,omitempty"`  //Change from open price to current (MID price)
	DAY_PERC_CHG_MID float64    `json:"DAY_PERC_CHG_MID,omitempty"` //Daily percentage change (MID price)
	DAY_HIGH         float64    `json:"DAY_HIGH,omitempty"`         //Daily high price (MID)
	DAY_LOW          float64    `json:"DAY_LOW,omitempty"`          //Daily low price (MID)
	OFR_OPEN         float64    `json:"OFR_OPEN,omitempty"`         //Candle open price (OFR)
	OFR_HIGH         float64    `json:"OFR_HIGH,omitempty"`         //Candle high price (OFR)
	OFR_LOW          float64    `json:"OFR_LOW,omitempty"`          //Candle low price (OFR)
	OFR_CLOSE        float64    `json:"OFR_CLOSE,omitempty"`        //Candle close price (OFR)
	BID_OPEN         float64    `json:"BID_OPEN,omitempty"`         //Candle open price (BID)
	BID_HIGH         float64    `json:"BID_HIGH,omitempty"`         //Candle high price (BID)
	BID_LOW          float64    `json:"BID_LOW,omitempty"`          //Candle low price (BID)
	BID_CLOSE        float64    `json:"BID_CLOSE,omitempty"`        //Candle close price (BID)
	LTP_OPEN         float64    `json:"LTP_OPEN,omitempty"`         //Candle open price (Last Traded Price)
	LTP_HIGH         float64    `json:"LTP_HIGH,omitempty"`         //Candle high price (Last Traded Price)
	LTP_LOW          float64    `json:"LTP_LOW,omitempty"`          //Candle low price (Last Traded Price)
	LTP_CLOSE        float64    `json:"LTP_CLOSE,omitempty"`        //Candle close price (Last Traded Price)
	CONS_END         float64    `json:"CONS_END,omitempty"`         //1 when candle ends, otherwise 0
	CONS_TICK_COUNT  float64    `json:"CONS_TICK_COUNT,omitempty"`  //Number of ticks in candle
}

func (dest *LightStreamChartTick) Merge(src *LightStreamChartTick) {
	if dest.LTV == 0 && src.LTV != 0 {
		(*dest).LTV = src.LTV
	}
	if dest.TTV == 0 && src.TTV != 0 {
		(*dest).TTV = src.TTV
	}
	if dest.UTM == nil && src.UTM != nil {
		(*dest).UTM = src.UTM
	}
	if dest.DAY_OPEN_MID == 0 && src.DAY_OPEN_MID != 0 {
		(*dest).DAY_OPEN_MID = src.DAY_OPEN_MID
	}
	if dest.DAY_NET_CHG_MID == 0 && src.DAY_NET_CHG_MID != 0 {
		(*dest).DAY_NET_CHG_MID = src.DAY_NET_CHG_MID
	}
	if dest.DAY_PERC_CHG_MID == 0 && src.DAY_PERC_CHG_MID != 0 {
		(*dest).DAY_PERC_CHG_MID = src.DAY_PERC_CHG_MID
	}
	if dest.DAY_HIGH == 0 && src.DAY_HIGH != 0 {
		(*dest).DAY_HIGH = src.DAY_HIGH
	}
	if dest.DAY_LOW == 0 && src.DAY_LOW != 0 {
		(*dest).DAY_LOW = src.DAY_LOW
	}
	if dest.OFR_OPEN == 0 && src.OFR_OPEN != 0 {
		(*dest).OFR_OPEN = src.OFR_OPEN
	}
	if dest.OFR_HIGH == 0 && src.OFR_HIGH != 0 {
		(*dest).OFR_HIGH = src.OFR_HIGH
	}
	if dest.OFR_LOW == 0 && src.OFR_LOW != 0 {
		(*dest).OFR_LOW = src.OFR_LOW
	}
	if dest.OFR_CLOSE == 0 && src.OFR_CLOSE != 0 {
		(*dest).OFR_CLOSE = src.OFR_CLOSE
	}
	if dest.BID_OPEN == 0 && src.BID_OPEN != 0 {
		(*dest).BID_OPEN = src.BID_OPEN
	}
	if dest.BID_HIGH == 0 && src.BID_HIGH != 0 {
		(*dest).BID_HIGH = src.BID_HIGH
	}
	if dest.BID_LOW == 0 && src.BID_LOW != 0 {
		(*dest).BID_LOW = src.BID_LOW
	}
	if dest.BID_CLOSE == 0 && src.BID_CLOSE != 0 {
		(*dest).BID_CLOSE = src.BID_CLOSE
	}
	if dest.LTP_OPEN == 0 && src.LTP_OPEN != 0 {
		(*dest).LTP_OPEN = src.LTP_OPEN
	}
	if dest.LTP_HIGH == 0 && src.LTP_HIGH != 0 {
		(*dest).LTP_HIGH = src.LTP_HIGH
	}
	if dest.LTP_LOW == 0 && src.LTP_LOW != 0 {
		(*dest).LTP_LOW = src.LTP_LOW
	}
	if dest.LTP_CLOSE == 0 && src.LTP_CLOSE != 0 {
		(*dest).LTP_CLOSE = src.LTP_CLOSE
	}
	if dest.CONS_END == 0 && src.CONS_END != 0 {
		(*dest).CONS_END = src.CONS_END
	}
	if dest.CONS_TICK_COUNT == 0 && src.CONS_TICK_COUNT != 0 {
		(*dest).CONS_TICK_COUNT = src.CONS_TICK_COUNT
	}
}

func NewLightStreamChartTick(fields []string, values []string) (*LightStreamChartTick, error) {
	if len(fields) != len(values) {
		return nil, fmt.Errorf("not enough values for fields number")
	}
	reflectType := reflect.TypeOf(LightStreamChartTick{})

	sliceToUnmarshal := make(map[string]interface{}, len(fields))
	for i := 0; i < len(fields); i++ {
		value := strings.ReplaceAll(values[i], "\r\n", "")
		if value == "" {
			continue
		}

		if reflectStructField, ok := reflectType.FieldByName(fields[i]); ok {
			t := reflectStructField.Type.Kind()
			switch t {
			case reflect.Float64:
				v, err := strconv.ParseFloat(value, 64)
				if err != nil {
					return nil, err
				}
				sliceToUnmarshal[fields[i]] = v
			case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
				v, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return nil, err
				}
				sliceToUnmarshal[fields[i]] = v
			case reflect.TypeOf(&time.Time{}).Kind():
				v, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return nil, err
				}
				sliceToUnmarshal[fields[i]] = time.Unix(0, v*int64(time.Millisecond))
			}
		}

	}

	bytesToUnmarshal, err := json.Marshal(sliceToUnmarshal)
	if err != nil {
		return nil, err
	}

	//fmt.Println(sliceToUnmarshal, string(bytesToUnmarshal))

	tick := LightStreamChartTick{}
	err = json.Unmarshal(bytesToUnmarshal, &tick)
	if err != nil {
		return nil, err
	}

	return &tick, nil
}

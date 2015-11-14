/*
Author: Vladimir Slav

This is free and unencumbered software released into the public domain.

Anyone is free to copy, modify, publish, use, compile, sell, or
distribute this software, either in source code form or as a compiled
binary, for any purpose, commercial or non-commercial, and by any
means.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS BE LIABLE FOR ANY CLAIM, DAMAGES OR
OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
OTHER DEALINGS IN THE SOFTWARE.
*/

package stockparser

import (
	"fmt"
	"net/http"
    "encoding/csv"
    "io"
    "strconv"
    "strings"
    "time"
    "math"
)

const DIVIDEND_LIST_FORMAT =     "http://real-chart.finance.yahoo.com/table.csv?s=%s&a=00&b=1&c=%d&d=12&e=31&f=%d&g=v&ignore=.csv";
const CLOSE_PRICE_LIST_FORMAT =  "http://real-chart.finance.yahoo.com/table.csv?s=%s&a=%d&b=%d&c=%d&d=%d&e=%d&f=%d&g=d&ignore=.csv";
//const DIVIDEND_LIST_FORMAT = "ichart.finance.yahoo.com/table.csv?s=YHOO&d=3&e=23&f=2010&g=d&a=3&b=12&c=1996&ignore=.csv";

type StockRecord struct {
    symbol string
    earnings float64
}

func getYearsDividend(yearLink string) [][]string {
    
    var result [][]string
	response, err := http.Get(yearLink);
	if err != nil {
        fmt.Printf("%s", err)
	} else {
        defer response.Body.Close()
        csvr:= csv.NewReader(response.Body)
        rec, err := csvr.Read()
        var first bool = true
        
        for err == nil {
            if (first) {
                first = false;
            } else {
                result = append(result, rec)
            }
            rec, err = csvr.Read()    
        }
        
        if err != io.EOF {
            print(err)
        }
	}
   

    return result
}

const DATE_YEAR_INDEX = 0;
const DATE_MONTH_INDEX = 1;
const DATE_DAY_INDEX = 2;

// Date,Open,High,Low,Close,Volume,Adj Close
const HISTORICAL_HIGH_INDEX = 2;

func summarizeData(buyPrice float64,
                   sellPrice float64,
                   dividendBonus float64,
                   budget float64,
                   commission float64) float64 {
                    
    var buyAmount int = int(math.Floor((float64(budget) - commission) / buyPrice));
    var buyMoneySpent float64 = float64(buyAmount) * buyPrice + commission;
    var sellMoneyGained float64 = float64(buyAmount) * sellPrice - commission;
    var dividendMoneyGained float64 = dividendBonus * float64(buyAmount);
    
    var balance float64 = dividendMoneyGained + sellMoneyGained - buyMoneySpent;
    // TODO: Output into a file in a separate function
    fmt.Printf(`BUYPRICE: %f,
               MONEY SPENT TO BUY: %f,
               AMOUNT BOUGHT: %d,
               SELLPRICE: %f,
               DIVIDEND PRICE: %f,
               AMOUNT GAINED FROM DIVIDENDS: %f,
               AMOUNT GAINED FROM SELL: %f,
               TOTAL PROFIT IN THE END OF THE TRANSACTION: %f`, 
               buyPrice,
               buyMoneySpent, 
               buyAmount,
               sellPrice,
               dividendBonus,
               dividendMoneyGained,
               sellMoneyGained,
               balance)     
    fmt.Println();
    return balance
}

func calculateDividendSum(symbol string,
                          dateUnprocessed string, 
                          dividendBonus float64,
                          daysBuyBefore,
                          daysSellAfter,
                          moneyForPurchase int,
                          commission float64) (result float64) {
    result = 0
    var dateSeparated []string = strings.Split(dateUnprocessed, "-")
    
    var year int64
    year, _ = strconv.ParseInt(dateSeparated[DATE_YEAR_INDEX], 10, 64)
    
    var month int64
    month, _ = strconv.ParseInt(dateSeparated[DATE_MONTH_INDEX], 10, 64)
    
    var day int64
    day, _ = strconv.ParseInt(dateSeparated[DATE_DAY_INDEX], 10, 64)

    var divDate time.Time = time.Date(int(year), time.Month(month), int(day), 0, 0, 0, 0, time.UTC)
    var buyDate time.Time = divDate.AddDate(0, 0, -daysBuyBefore)
    var sellDate time.Time = divDate.AddDate(0, 0, daysSellAfter)
    
    var datalink string
    openYear, openMonth, openDay := buyDate.Date()
    closeYear, closeMonth, closeDay := sellDate.Date()
    
    datalink = fmt.Sprintf(CLOSE_PRICE_LIST_FORMAT, symbol, int(openMonth) - 1, openDay, openYear, int(closeMonth - 1), closeDay, closeYear)
	response, err := http.Get(datalink);
    fmt.Println("========");
    fmt.Println("Getting Data from: " + datalink + " , dividend date: " + divDate.String());
	if err != nil {
        fmt.Printf("%s", err)
	} else {
        defer response.Body.Close()
        csvr:= csv.NewReader(response.Body)
        rec, err := csvr.ReadAll()
        
        if (err == nil) {
            var buyCost float64 = 0
            var sellCost float64 = 0;
            buyCost, errBuy := strconv.ParseFloat(rec[1][HISTORICAL_HIGH_INDEX], 64);
            sellCost, errSell := strconv.ParseFloat(rec[len(rec) - 1][HISTORICAL_HIGH_INDEX], 64)
            if errBuy == nil && errSell == nil {
                
                result = result + summarizeData(buyCost, sellCost, dividendBonus, float64(moneyForPurchase), commission)
            }
            
        }
	}
    //time.Time buyDate = divDate.
        
    
    return result
}

const DATE_INDEX int = 0
const DIVIDEND_INDEX int = 1

func ReadDividendData(symbol string,
                      yearStart int,
                      yearEnd int,
                      baseMoney int,
                      daysBuyBefore int,
                      daysSellAfter int,
                      commission float64) StockRecord {
    var sum float64 = 0;
    
    var link_str string
    link_str = fmt.Sprintf(DIVIDEND_LIST_FORMAT, symbol, yearStart, yearEnd)
    
    var date_dividend [][]string = getYearsDividend(link_str)
    
    for line := range date_dividend {
        var divPrice float64 = 0
        divPrice, err := strconv.ParseFloat(date_dividend[line][1], 64)
        if (err == nil) {
            sum += calculateDividendSum(symbol, date_dividend[line][0], divPrice, daysBuyBefore, daysSellAfter, baseMoney, commission)
        } else {
            fmt.Println(err.Error())
        }
    }
    
    sum += float64(len(date_dividend))
    return StockRecord{symbol, sum};    
}

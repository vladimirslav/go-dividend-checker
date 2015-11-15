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
    "os"
)

const DIVIDEND_LIST_FORMAT =     "http://real-chart.finance.yahoo.com/table.csv?s=%s&a=00&b=1&c=%d&d=12&e=31&f=%d&g=v&ignore=.csv";
const CLOSE_PRICE_LIST_FORMAT =  "http://real-chart.finance.yahoo.com/table.csv?s=%s&a=%d&b=%d&c=%d&d=%d&e=%d&f=%d&g=d&ignore=.csv";

type StockRecord struct {
    Symbol string
    Earnings float64
}

func getYearsDividend(yearLink string) [][]string {
    
    var result [][]string
    response, err := http.Get(yearLink);
    if err != nil {
        fmt.Println(err)
    } else {
        defer response.Body.Close()
        csvr:= csv.NewReader(response.Body)
        rec, err := csvr.Read()
        var first bool = true
        
        for err == nil {
            if (first) {
                // ignore first line as it contains the names of the columns
                // but not the data
                first = false;
            } else {
                result = append(result, rec)
            }
            rec, err = csvr.Read()    
        }
        
        if err != io.EOF {
            fmt.Println(err)
        }
    }
   

    return result
}

func summarizeData(buyPrice float64,
                   sellPrice float64,
                   dividendBonus float64,
                   budget float64,
                   commission float64,
                   f *os.File) float64 {
                    
    // how many stock can we buy with money we have? 
    // substract commission from our budget and divide by price of one stock
    var buyAmount int = int(math.Floor((float64(budget) - commission) / buyPrice));
    
    // how much money we actually spent? 
    // they can be leftover money if we had budget if we have
    // for example 40 USD in remaining budget and one stock price is 30 USD
    var buyMoneySpent float64 = float64(buyAmount) * buyPrice + commission;
    
    // how much money do we get from selling by sellPrice 
    // after we get the dividend? substract commission 
    // because it is charged on sell operations too
    var sellMoneyGained float64 = float64(buyAmount) * sellPrice - commission;
    
    // how much money did we actually get from dividend?
    // multiply stock amount with dividend payout for one stock
    var dividendMoneyGained float64 = dividendBonus * float64(buyAmount);
    
    // calculate final balance
    var balance float64 = dividendMoneyGained + sellMoneyGained - buyMoneySpent;
    
    // log into file
    // Proper way would be to make one more function
    // But since this is a weekend project
    // I try to forgive myself
    var logData = fmt.Sprintf(`BUYPRICE: %f,
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
    f.WriteString(logData);
    f.WriteString("");
    
    return balance
}

const DATE_YEAR_INDEX = 0;
const DATE_MONTH_INDEX = 1;
const DATE_DAY_INDEX = 2;

const HISTORICAL_HIGH_INDEX = 2;

func calculateDividendSum(symbol string,
                          dateUnprocessed string, 
                          dividendBonus float64,
                          daysBuyBefore,
                          daysSellAfter,
                          moneyForPurchase int,
                          commission float64,
                          f *os.File) (result float64) {
    result = 0
    // date is passed as yyyy-mm-dd
    // first explode it into logical chunks, removing "-" and placing
    // year, month and day into separate indexes of a string array
    var dateSeparated []string = strings.Split(dateUnprocessed, "-")
    
    var year int64
    year, _ = strconv.ParseInt(dateSeparated[DATE_YEAR_INDEX], 10, 64)
    
    var month int64
    month, _ = strconv.ParseInt(dateSeparated[DATE_MONTH_INDEX], 10, 64)
    
    var day int64
    day, _ = strconv.ParseInt(dateSeparated[DATE_DAY_INDEX], 10, 64)

    // calculate the dividend date
    var divDate time.Time = time.Date(int(year), time.Month(month), int(day), 0, 0, 0, 0, time.UTC)
    // then substract given amount of days to get the date when we purchase the stock
    var buyDate time.Time = divDate.AddDate(0, 0, -daysBuyBefore)
    // then add given amount of days to get the date when we sell the stock
    var sellDate time.Time = divDate.AddDate(0, 0, daysSellAfter)
    
    var datalink string
    openYear, openMonth, openDay := buyDate.Date()
    closeYear, closeMonth, closeDay := sellDate.Date()
    
    // form the proper link to request a historical dates between buy and sell date
    datalink = fmt.Sprintf(CLOSE_PRICE_LIST_FORMAT,
                           symbol,
                           int(openMonth) - 1,
                           openDay,
                           openYear,
                           int(closeMonth - 1),
                           closeDay,
                           closeYear)
                        
    response, err := http.Get(datalink);
    fmt.Println("========");
    fmt.Println("Getting Data from: " + datalink + " , dividend date: " + divDate.String());
    
    if err != nil {
        fmt.Println(err)
    } else {
        defer response.Body.Close()
        // the csv format:
        // Date,Open,High,Low,Close,Volume,Adj Close
        csvr:= csv.NewReader(response.Body)
        rec, err := csvr.ReadAll()
        
        if (err == nil) {
            f.WriteString("===========\n");
            f.WriteString("Getting Data from: " + datalink + " , dividend date: " + divDate.String())
            f.WriteString("\n")

            var buyCost float64 = 0
            var sellCost float64 = 0
            
            // buy price is at the bottom of the list (earliest date)
            // respectuflly, sell price is on top (we sell for that price)
            buyCost, errBuy := strconv.ParseFloat(rec[len(rec) - 1][HISTORICAL_HIGH_INDEX], 64);
            sellCost, errSell := strconv.ParseFloat(rec[1][HISTORICAL_HIGH_INDEX], 64)
            if errBuy == nil && errSell == nil && err == nil {
                
                result = summarizeData(buyCost, sellCost, dividendBonus, float64(moneyForPurchase), commission, f)
            }                
        }
    }
    
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
                        
    // this will store the sum of all possible trades for that symbol
    // that were made during the given time frame
    var sum float64 = 0;
    
    var link_str string
    // form the link - insert the value
    link_str = fmt.Sprintf(DIVIDEND_LIST_FORMAT, symbol, yearStart, yearEnd)
    
    // get the array of dividends - dates/amounts paid
    // yahoo finance csv format requested by format link
    // just includes the date and dividend in one string
    var date_dividend [][]string = getYearsDividend(link_str)
    
    f, err:= os.Create("res/" + symbol + ".txt")    
    if err == nil {
        defer f.Close()
        for line := range date_dividend {
            var divPrice float64 = 0
            divPrice, err := strconv.ParseFloat(date_dividend[line][DIVIDEND_INDEX], 64)
            if (err == nil) {
                if err == nil {
                    sum += calculateDividendSum(symbol,
                                                date_dividend[line][DATE_INDEX],
                                                divPrice,
                                                daysBuyBefore,
                                                daysSellAfter,
                                                baseMoney,
                                                commission,
                                                f)
                }
            } else {
                f.WriteString(err.Error())
            }
        }
    }
    
    f.WriteString("\n=========\n")
    f.WriteString("Final Sum: " + strconv.FormatFloat(sum, 'f', 3, 64))
    
    return StockRecord{symbol, sum};    
}

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

package main

import (
    "fmt"
    "stockparser"
    "os"
    "time"
    "encoding/csv"
    "io"
    "sort"
    "strconv"
)

type byEarnings []stockparser.StockRecord
func (arr byEarnings) Len() int { return len(arr) }
func (arr byEarnings) Swap(i, j int) { arr[i], arr[j] = arr[j], arr[i] }
func (arr byEarnings) Less(i, j int) bool { return arr[i].Earnings < arr[j].Earnings } 

func main() {
    csvfile, err:= os.Open("companylist.csv")
    
    if err != nil {
        fmt.Println(err)
        return
    }
    
    defer csvfile.Close()
    
    var records byEarnings
    
    reader := csv.NewReader(csvfile)
    rec, err := reader.Read()
    var first bool = true
    var counter int = 0
    fmt.Println("Reading data...")
    for err == nil {
        if (first) {
            first = false;
        } else {
            fmt.Println("Reading line " + strconv.Itoa(counter))
            records = append(records, stockparser.ReadDividendData(rec[0], 2012, 2015, 4000, 5, 5, 18));
            time.Sleep(5000 * time.Millisecond)
        }
        rec, err = reader.Read()
        counter++
    }
    
    if err != io.EOF {
        print(err)
    }
    
    fmt.Println("Sorting Data")
    sort.Sort(records)
    fmt.Println("Done")
    
    for _,element := range records {
        fmt.Println("S: " + element.Symbol + " Earnings: " + strconv.FormatFloat(element.Earnings, 'f', 3, 64))
    }
    
}

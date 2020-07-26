# Flux

![Flux Logo](http://img.aditya.diwakar.io/FluxLogo.png)

## TDAmeritrade Streamer Documentation

The sparsely documented streamer websocket from TDAmeritrade is a feature of the OAuth Apps that are available in their TDAmeritrade Developer Center, authentication for this task is simply done by using your refresh token and consumer key to generate the access token for the intial login. The entire websocket is documented here, in accordance.

### Messages

TDAmeritrade has opted to utilize the JSON content type for every message, and respects that. If an invalid object is pushed into the socket, they return an error accordingly. On top of that, other than the initial login and protocol configuration messages, the socket uses **json-patch** (see [RFC 6902](https://tools.ietf.org/html/rfc6902)) in order to modify previously received messages.

This also means that making the same request twice will result in no data being sent, since the socket expects you to cache all data. Flux utilizes this in a peculiar fashion by storing more than just the latest message, but all relevant messages (on an ad-hoc per service basis). This is demonstrated in the Session's structure, seen here:

```go
type Session struct {
    TdaSession         tda.Session
    wsConn             *websocket.Conn
    ConfigUrl          string
    CurrentState       []byte
    CurrentChartHash   string
    TransactionChannel chan cachedData
}
```

The ``CurrentState`` byte slice stores relevant data for all services; this is just a marshalled object (stored in a byte slice for expandability). If the data being pulled using Flux was both a chart object and an instrument details object, the CurrentSlice (stringly typed) would like so:

```json
{
    "chart": ...,
    "instrument_details": ...
}
```

As a result, any incoming json patches are manipulated with their data paths modified to respect this behavior. If you are writing your own library, this is important to note that the data is expected to be stored on the client side.

#### Message Formats

Every single request that the client will send to the server must be formatted by being wrapped in a ``payload`` array, this is done so that the client can send multiple requests at the same time to the server (which is incredibly useful for subscribing to tickers in a watchlist, for example). Every message will be some object (with very similar structure), but will be sent into the stream as:

```json
{
    "payload": [
        REQUEST_OBJECTS...
    ]
}
```

**Note:** If your request object utilizes the `id` field, this can be used to match a request and a response. For example, if I specify the id as ``abc123``, the response that correlates to that respond will be ``abc123`` which can be used for matching.

### Handshake

The intial job when connecting to the server is to properly connect using the handshake, this is done in 4 steps.

1. Make the connection with the socket, the gateway URL can be received from https://trade.thinkorswim.com/v1/api/config under the ``livetrading`` field.
2. Send the protocol message (and ensure the websocket was not closed upon dialup) setting the version, format, and heartbeat frequency accordingly. This can be seen in Flux [here](https://github.com/adityaxdiwakar/flux/blob/master/conn.go#L66) here. The protocol message would need to be:

```json
{
    "ver": "25.*.*",
    "fmt": "json-patches",
    "heartbeat": "2s"
}
```

The version number above is bound to change and will be periodically updated here when that number does change (and causes an invalidated session). On top of that, theheartbeat frequency is up to you if you are designing your own lib, ``2s`` is standard but it can be set to any frequency. Our recommendations is to not use anything longerthan ``5s``.

Note: You are not responsible for acknowledging or sending a heartbeat back, the heartback is only sent from TDAmeritrade, sending one back will cause an error.

3. After dialing up the connection and sending the initial message, you will need to wait for a message to come from TDAmeritrade which contains some protocol information, this message can be ignored (just ensure that it does come, before continuing to authentication). The message will look like so:

```json
{"session":"SOME_SESSION_ID","build":"26.2029.0-N","ver":"25.*.*"}
```

The session ID is an alphanumeric string that is used to identify your connection on TDAmeritrade's Java servers. This does not need to be used client side, and therefore canbe ignored.

4. The final step of the handshake is authentication. In order to authenticate, the client needs to send a JSON object containing authentication credentials, this object is designed like so:

```json
{
  "service": "login",
  "id": "login",
  "ver": 0,
  "domain": "TOS",
  "platform": "PROD",
  "token": "",
  "accessToken": "YOUR_ACCESS_TOKEN",
  "tag": "TOSWeb"
}
```

The ``accessToken`` should be populated by using your access token (generated using your access and refresh token).
In order to send this authentication, wrap it as mentioned [earlier](#message-formats) and write it into the connection.

After completing the handshake, the socket will return a message regarding your rights and privileges (such as market and trading permissions). This message is not entirely important, however, it does return a `token` string which can be used for further authentication (use it in the `token` field, rather than the `accessToken` field).

## Charts

### Request

Charts are quite simple, you can retrieve a chart using the following object:

```json
{
  "service": "chart",
  "id": "UNIQUE_ID",
  "ver": 0,
  "symbol": "TICKER",
  "aggregationPeriod": "AGGREGATION PERIOD",
  "studies": [],
  "range": "RANGE"
}
```

The `symbol` field can be filled in with the ticker, these are the same as the ones you would find on ThinkOrSwim, for example: ``/ES`` for E-Mini S&P 500 Futures, ``SPX`` for the S&P 500 index, and ``SPY`` for the S&P 500 index ETF.

The ``aggregationPeriod`` and ``range`` are two fields that can be found in the spec sheet [here](https://github.com/adityaxdiwakar/flux/blob/master/specs.txt), they define the timeframe and the width of the candles. Remember, sending this message requires you to wrap it in a payload array, as mentioned [earlier](#message-formats).

### Response

There are two types of responses for the chart object. The first is the initial chart response, and the second is a candle updater (to keep the candles up to date).

Initial Chart Response:

```json
{
  "op": "replace",
  "path": "",
  "value": {
    "symbol": "AAPL",
    "instrument": {
      "symbol": "AAPL",
      "rootSymbol": "AAPL",
      "displaySymbol": "AAPL",
      "rootDisplaySymbol": "AAPL",
      "futureOption": false,
      "description": "APPLE INC COM",
      "multiplier": 1,
      "spreadsSupported": true,
      "tradeable": true,
      "instrumentType": "STOCK",
      "id": 18560,
      "sourceType": "NASDAQ",
      "isFutureProduct": false,
      "hasOptions": true,
      "composite": false,
      "fractionalType": "X10",
      "daysToExpiration": 0,
      "spreadDaysToExpiration": "0",
      "cusip": "037833100",
      "industry": 124,
      "spc": 1,
      "extoEnabled": false,
      "flags": 0
    },
    "timestamps": [],
    "open": [],
    "high": [],
    "low": [],
    "close": [],
    "volume": [],
    "service": "chart",
    "requestId": "UNIQUE_ID",
    "requestVer": 0
  }
}
```

The above is an example for a chart request for ``AAPL``, the arrays near the bottom (OHLC + time) are ommited but will be populated with numbers accordingly, which can be used to make OHLC candles through whatever library you may be using.

Subsequent responses for this service will be displayed as patches, occassionally replacing the last index for the close and volume arrays, and adding new timestamps for the creation of new candles, this has now become a subscribed chart.

## Instrument Search

### Request

Searching for instruments is a pretty powerful feature for finding the charts and information for tickers you may not know. A search can be executed with the following object:

```json
{
  "service": "instrument_search",
  "limit": 5,
  "pattern": "SEARCH_KEY",
  "ver": 0,
  "id": "UNIQUE_ID"
}
```

The ``pattern`` field can be anything that you would like, for example; if you know the beginning of a ticker, you can use that and use the resulting list accordingly. On top of that, you can search for tickers by using the company name. For example, if you would like to search for Microsoft's ticker, use "Microsoft" as the pattern.

### Response

The response to a search query is the following object (example used is searching for `Semiconductor`):

```json
{
  "op": "replace",
  "path": "",
  "value": {
    "instruments": [
      {
        "symbol": "TSM",
        "rootSymbol": "TSM",
        "displaySymbol": "TSM",
        "rootDisplaySymbol": "TSM",
        "futureOption": false,
        "description": "TAIWAN SEMICONDUCTOR MANUFACTU ADR SPONSORED",
        "multiplier": 1,
        "spreadsSupported": true,
        "tradeable": true,
        "instrumentType": "STOCK",
        "id": 35039,
        "sourceType": "NYSE",
        "isFutureProduct": false,
        "hasOptions": true,
        "composite": false,
        "fractionalType": "X10",
        "daysToExpiration": 0,
        "spreadDaysToExpiration": "0",
        "cusip": "874039100",
        "industry": 130,
        "spc": 1,
        "extoEnabled": false,
        "flags": 0
      },
      ...
      ...
      ...
      ...
    ],
    "service": "instrument_search",
    "requestId": "UNIQUE_ID",
    "requestVer": 0
  }
}
```

Only one entry is shown, although there would be 5 total listed (as evident by the ...).

## Instrument Details

### Request

The example below uses `BMY` as the ticker

```json
{
  "service": "instrument_details",
  "id": "instrument/BMY",
  "ver": 0,
  "symbol": "BMY",
  "fields": [
    "DIV_AMOUNT",
    "EPS",
    "EXD_DIV_DATE",
    "PRICE_TICK",
    "PRICE_TICK_VALUE",
    "YIELD",
    "DIV_FREQUENCY"
  ]
}
```

**Note:** The above `fields` are all the fields available to be used, if you want fewer fields while making a request, feel free to remove them. However, as a disclaimer, please note that TDAmeritrade is a bit unreliable for some of these numbers, and they may or may not be populated consistently.

### Response

The response from the server is below (example is `AAPL`):

```json
{
  "op": "replace",
  "path": "",
  "value": {
    "instrument": {
      "symbol": "AAPL",
      "rootSymbol": "AAPL",
      "displaySymbol": "AAPL",
      "rootDisplaySymbol": "AAPL",
      "futureOption": false,
      "description": "APPLE INC COM",
      "multiplier": 1,
      "spreadsSupported": true,
      "tradeable": true,
      "instrumentType": "STOCK",
      "id": 18560,
      "sourceType": "NASDAQ",
      "isFutureProduct": false,
      "hasOptions": true,
      "hasTradableOptions": true,
      "composite": false,
      "fractionalType": "X10",
      "daysToExpiration": 0,
      "spreadDaysToExpiration": "0",
      "cusip": "037833100",
      "industry": 124,
      "spc": 1,
      "extoEnabled": false,
      "flags": 0,
      "priceTick": 0.01,
      "priceTickValue": 0.01
    },
    "values": {
      "DIV_AMOUNT": 0.82,
      "EPS": 12.79,
      "EXD_DIV_DATE": 1588939200000,
      "PRICE_TICK": 0.01,
      "PRICE_TICK_VALUE": 0.01,
      "YIELD": 0.0089
    },
    "service": "instrument_details",
    "requestId": "instrument/AAPL",
    "requestVer": 0
  }
}
```

If the value does not exist for some stock (i.e pulling dividend information from `AMZN`, which does not pay a dividend), the value will not exist. Flux handles this with null empty ommited fields in the relevant structs.

# goweather
Go REST Weather Web Service

## Description

This is a simple Go program that lets you run a REST API webservice.
It expose the `/weather/` api that can be queried just by concatenating it with the name of the place you want the weather for the moment you ask.

 - Example : `/weather/london`
 - Response : `{"city":"london","temp":277.53499999999997,"took":"229.034322ms"}`

This service relies on OpenWeatherMap and Weather Underground APIs: their temperatures are averaged and sent in the response to have a better approximation.

Analyze the JSON response:

- `city` : the city that follows `/weather/`
- `temp` : the current temperature in Kelvin
- `took` : amount of time elapsed in ms

## Installation and usage

- `git clone https://github.com/indiependente/goweather.git`
- `cd goweather`
- `go build`
- `./goweather`
- `curl localhost:8080/weather/london`

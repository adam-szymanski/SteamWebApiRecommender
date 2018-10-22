# Steam web crawler and item-item recommender system

## Overview

This is a prototype. It consist several components that are used to build working solution.

## Components

### User crawler

This is placed in user_crawler/user_crawler.go. It is crawling Steam user profiles and places them in dump folder. To use it please put your SteamWebAPI key in apiKey const.

### User updater

Updates user profiles stored in data/users.binary using data from dump folder crawled by user_crawler. User profile contain last value of time spend in games.

### Data processor

Reads user profiles stored in data/users.binary and calculate intermediate data and recommendations. Result is stored in data/products_with_stats.binary.

### Products crawler

Reads data/products_with_stats.binary, looks for products without Steam Store descriptions and parses it from Steam web page. Read data is stored in dump folder.

### Product updater

Reads data/products_with_stats.binary and Steam Store data from dump, adds data from dump to enrich data in data/products_with_stats.binary. Result is stored in data/products_with_stats_and_market.binary.

### Prepare serving data

Reads data/products_with_stats_and_market.binary, removes unnecessary data, compiles to form easier for serving and writes in data/serving_data.binary.

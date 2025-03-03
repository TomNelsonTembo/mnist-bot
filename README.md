# Automated Request Mnist-Bot for Model Testing

This repository contains the source code for an Automated Request Bot developed as part of a Bachelor's Thesis Project. The bot is designed to simulate real-world usage of machine learning models by sending controlled HTTP requests to a deployed model's API endpoint. It is written in Go and leverages Go's concurrency model to efficiently handle multiple requests simultaneously. The bot was used to evaluate the performance of an optimized Convolutional Neural Network (CNN) model, focusing on metrics such as latency, throughput, and scalability.

The bot was instrumental in benchmarking the model's performance under various conditions, including low, medium, and high traffic scenarios, as well as edge computing environments. It was integrated with TensorFlow Serving for efficient model serving.

## Features
- Customizable Request Rate: Adjust the rate at which requests are sent to simulate different traffic conditions.
- Concurrent Requests: Utilizes Go's goroutines to send multiple requests concurrently, mimicking real-world usage patterns.

## Prerequisites
Before running the bot, ensure you have the following installed:
- Go (version 1.20 or higher)
- Git (for cloning the repository)
- TensorFlow Serving (for serving the model)

## Installation
1. Clone the repository:
```
git clone git@github.com:TomNelsonTembo/mnist-bot.git
cd mnist-bot
```

2. Install dependencies:
```
go mod tidy
```

3. Build the bot:
```
go build -o mnist-bot
```

## Usage
To run the bot, use the following command:
```
./mnist-bot.exe --api=<API_ENDPOINT> --interval <REQUEST_INTERVAL> --bots <NUMBER_OF_CONCURRENT_REQUESTS> --data ./Assets/Data/data.json
```

## Contribution
This project was developed as part of a Bachelor's Thesis titled "Optimizing Cloud-Based Machine Learning Models for Low-Latency Applications". Contributions to the project are welcome. If you find any issues or have suggestions for improvements, please open an issue or submit a pull request.

## Acknowledgments
- Go Programming Language: For its efficient concurrency model and lightweight design.
- TensorFlow Serving: For providing a scalable and flexible serving system for machine learning models.

For any questions or further information, please contact the author at [Tom Nelson Tembo](https://www.linkedin.com/in/tom-nelson-tembo-440ba0235/).

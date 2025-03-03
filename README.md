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
git clone https://github.com/your-username/automated-request-bot.git
cd automated-request-bot
```

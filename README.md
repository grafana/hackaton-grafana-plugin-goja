# Hackathon: Grafana frontend datasource running in the backend

This is an experimental datasouce made as a hackathon project.

This datasource is technically written in frontend as in, the logic that fetches and return data (from json placeholder API) is written in javascript, with the exact same code a regular grafana frontend datasource would do.

There's special code in the frontend module.ts that returns a proxy datasource that makes grafana believe this is a backend data source instead.

This datasource also has a backend component that runs a nodejs process with the frontend datasource and proxy queries to it. This backend component also has logic to convert backend queries into frontend queries and frontend data frames into backend data frames.

There's a custom wasm.ts script that works as a init script for the nodejs process. It initializes the datasource and makes it ready for querying.

Here's an overview of how the plugin works:

![image](https://github.com/user-attachments/assets/35e7f686-4bda-4f66-aa52-fb9bde251915)

## Is the go code necessary?

For this specific code yes, because the go code is working as an intermediary between Grafana and nodejs. Go here is providing a GRPC server and Apache arrow serialization for responses. If one were to implement the grpc and apache arrow logic in nodejs the plugin could run entirely in nodejs

## How to run this?

### Requirements

- Go
- Mage
- Docker
- Nodejs

### Steps

- clone the repository
- run `npm install`
- run `npm run build`
- run `cp ./dist/module.js pkg/plugin/module.js` (the module.js is embbeded in the go binary)
- run `mage -v build:backend`
- run `npm run server`

Once grafana started inside docker you can viist `http://localhost:3000` and create a new dashboard selecting "wasm" as the datasource

## Why is this called WASM datasource? where's the WASM part?

The original project wanted to use wasm to run the frontend code but heavy limitations on how frontend code can run inside a wasm environment with a js engine made the project impossible to complete in the time frame.

The current code might mention wasm somewhere or even include related dependencies (need clean up) but wasm is not used anywhere and it is not a part of the project.

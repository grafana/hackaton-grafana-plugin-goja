const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');
const path = require('path');

const PROTO_PATH = path.resolve(__dirname, 'backend.proto');

const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
});

const protoDescriptor = grpc.loadPackageDefinition(packageDefinition);
const pluginv2 = protoDescriptor.pluginv2;

// Data Service
const dataService = {
  queryData: (call, callback) => {
    const response = {
      responses: {
        A: {
          frames: [],
          status: 200,
        },
      },
    };
    callback(null, response);
  },
};

// Resource Service
const resourceService = {
  callResource: (call) => {
    call.end();
  },
};

// Diagnostics Service
const diagnosticsService = {
  checkHealth: (call, callback) => {
    callback(null, {
      status: 1, // OK
      message: 'OK',
    });
  },
  collectMetrics: (call, callback) => {
    callback(null, {});
  },
};

// Stream Service
const streamService = {
  subscribeStream: (call, callback) => {
    callback(null, { status: 0 });
  },
  runStream: (call) => {
    call.end();
  },
  publishStream: (call, callback) => {
    callback(null, { status: 0 });
  },
};

// AdmissionControl Service
const admissionControlService = {
  validateAdmission: (call, callback) => {
    callback(null, { allowed: true });
  },
  mutateAdmission: (call, callback) => {
    callback(null, { allowed: true });
  },
};

// ResourceConversion Service
const resourceConversionService = {
  convertObjects: (call, callback) => {
    callback(null, { uid: call.request.uid });
  },
};

function startServer() {
  const server = new grpc.Server();

  server.addService(pluginv2.Data.service, dataService);
  server.addService(pluginv2.Resource.service, resourceService);
  server.addService(pluginv2.Diagnostics.service, diagnosticsService);
  server.addService(pluginv2.Stream.service, streamService);
  server.addService(pluginv2.AdmissionControl.service, admissionControlService);
  server.addService(pluginv2.ResourceConversion.service, resourceConversionService);

  server.bindAsync('0.0.0.0:0', grpc.ServerCredentials.createInsecure(), (error, port) => {
    if (error) {
      console.error(error);
      return;
    }
    // write port to `./dist/standalone.txt`
    require('fs').writeFileSync('./dist/standalone.txt', `:${port}`);
    console.log(port);
  });
}

startServer();

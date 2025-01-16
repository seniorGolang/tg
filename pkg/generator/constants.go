// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (constants.go at 19.06.2020, 16:08) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

const (
	syncHeader            = "X-Sync-On"
	packageOS             = "os"
	packageIO             = "io"
	_ctx_                 = "ctx"
	packageFmt            = "fmt"
	packageTLS            = "crypto/tls"
	packageTime           = "time"
	_next_                = "next"
	packageSync           = "sync"
	packageTesting        = "testing"
	packageReflect        = "reflect"
	packageHttp           = "net/http"
	packageContext        = "context"
	packageStrconv        = "strconv"
	packageStrings        = "strings"
	packageStdJSON        = "encoding/json"
	packageMultipart      = "mime/multipart"
	packageCors           = "github.com/lab259/cors"
	packageErrors         = "github.com/pkg/errors"
	packageUUID           = "github.com/google/uuid"
	packageFiber          = "github.com/gofiber/fiber/v2"
	packageZeroLog        = "github.com/rs/zerolog"
	packageZeroLogLog     = "github.com/rs/zerolog/log"
	packageFiberAdaptor   = "github.com/gofiber/adaptor/v2"
	packageAttributeOTEL  = "go.opentelemetry.io/otel/attribute"
	packageOTEL           = "go.opentelemetry.io/otel"
	packageTrace          = "go.opentelemetry.io/otel/trace"
	packageFasthttp       = "github.com/valyala/fasthttp"
	packagePrometheus     = "github.com/prometheus/client_golang/prometheus"
	packagePrometheusAuto = "github.com/prometheus/client_golang/prometheus/promauto"
	packagePrometheusHttp = "github.com/prometheus/client_golang/prometheus/promhttp"
)

const jsonRPCClientBase = `
export class JSONRPCError extends Error {
	constructor(message, name, code, data) {
	  	super(message);
	  	this.name = name;
	  	this.code = code;
		this.data = data;
	}
}

class JSONRPCScheduler {
	/**
	 *
	 * @param {*} transport
	 */
	constructor(transport) {
	  this._transport = transport;
	  this._requestID = 0;
	  this._scheduleRequests = {};
	  this._commitTimerID = null;
	  this._beforeRequest = null;
	}
	beforeRequest(fn) {
	  this._beforeRequest = fn;
	} 
	__scheduleCommit() {
	  if (this._commitTimerID) {
		clearTimeout(this._commitTimerID);
	  }
	  this._commitTimerID = setTimeout(() => {
		this._commitTimerID = null;
		const scheduleRequests = { ...this._scheduleRequests };
		this._scheduleRequests = {};
		let requests = [];
		for (let key in scheduleRequests) {
		  requests.push(scheduleRequests[key].request);
		}
		this.__doRequest(requests)
		  .then((responses) => {
			for (let i = 0; i < responses.length; i++) {
              const schedule = scheduleRequests[responses[i].id];
			  if (responses[i].error) {
				schedule.reject(responses[i].error);
				continue;
			  }
			  schedule.resolve(responses[i].result);
			}
		  })
         .catch((e) => {
           for (let key in requests) {
             if (!requests.hasOwnProperty(key)) {
               continue;
             }
             if (scheduleRequests.hasOwnProperty(requests[key].id)) {
               scheduleRequests[requests[key].id].reject(e)
             }
           }
         });
	  }, 0);
	}
	makeJSONRPCRequest(id, method, params) {
	  return {
		jsonrpc: "2.0",
		id: id,
		method: method,
		params: params,
	  };
	}
	/**
    * @param {string} method
    * @param {Object} params
    * @returns {Promise<*>}
    */
	__scheduleRequest(method, params) {
	  const p = new Promise((resolve, reject) => {
		const request = this.makeJSONRPCRequest(
		  this.__requestIDGenerate(),
		  method,
		  params
		);
		this._scheduleRequests[request.id] = {
		  request,
		  resolve,
		  reject,
		};
	  });
	  this.__scheduleCommit();
	  return p;
	}
	__doRequest(request) {
	  return this._transport.doRequest(request);
	}
	__requestIDGenerate() {
	  return ++this._requestID;
	}
 }
`

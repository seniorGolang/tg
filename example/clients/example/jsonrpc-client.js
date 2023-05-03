
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
class JSONRPCClientExampleRPC {
constructor(transport) {
this.scheduler = new JSONRPCScheduler(transport);
}

/**
* json RPC метод
*
* @param {string} Arg1
* @param {...Array<>} Opts
* @return {PromiseLike<{Ret1: number,Ret2: string}>}
**/
test(arg1,...opts) {
return this.scheduler.__scheduleRequest("exampleRPC.test", {arg1:arg1,opts:opts}).catch(e => { throw exampleRPCTestConvertError(e); })
}
/**
* @param {number} Arg0
* @param {string} Arg1
* @param {...Array<>} Opts
* @return {PromiseLike<{Ret1: number,Ret2: string}>}
**/
test2(arg0,arg1,...opts) {
return this.scheduler.__scheduleRequest("exampleRPC.test2", {arg0:arg0,arg1:arg1,opts:opts}).catch(e => { throw exampleRPCTest2ConvertError(e); })
}
}

class JSONRPCClient {
constructor(transport) {
this.exampleRPC = new JSONRPCClientExampleRPC(transport);
}
}
export default JSONRPCClient

function exampleRPCTestConvertError(e) {
switch(e.code) {
default:
return new JSONRPCError(e.message, "UnknownError", e.code, e.data);
}
}
function exampleRPCTest2ConvertError(e) {
switch(e.code) {
default:
return new JSONRPCError(e.message, "UnknownError", e.code, e.data);
}
}
/**
* @typedef {interface} interface
*/


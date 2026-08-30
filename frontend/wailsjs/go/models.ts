export namespace main {
	
	export class PortInfo {
	    port: number;
	    pid: number;
	    procName: string;
	    project: string;
	
	    static createFrom(source: any = {}) {
	        return new PortInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.port = source["port"];
	        this.pid = source["pid"];
	        this.procName = source["procName"];
	        this.project = source["project"];
	    }
	}
	export class TOTPCode {
	    name: string;
	    issuer: string;
	    code: string;
	    remain: number;
	
	    static createFrom(source: any = {}) {
	        return new TOTPCode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.issuer = source["issuer"];
	        this.code = source["code"];
	        this.remain = source["remain"];
	    }
	}
	export class TOTPSecret {
	    name: string;
	    secret: string;
	    issuer: string;
	
	    static createFrom(source: any = {}) {
	        return new TOTPSecret(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.secret = source["secret"];
	        this.issuer = source["issuer"];
	    }
	}

}


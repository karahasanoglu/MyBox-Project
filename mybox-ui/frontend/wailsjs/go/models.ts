export namespace main {
	
	export class BuildImageReq {
	    tag: string;
	    context: string;
	
	    static createFrom(source: any = {}) {
	        return new BuildImageReq(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tag = source["tag"];
	        this.context = source["context"];
	    }
	}
	export class RunContainerReq {
	    image: string;
	    command: string[];
	    ports: string;
	    memory: string;
	    cpus: string;
	
	    static createFrom(source: any = {}) {
	        return new RunContainerReq(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.image = source["image"];
	        this.command = source["command"];
	        this.ports = source["ports"];
	        this.memory = source["memory"];
	        this.cpus = source["cpus"];
	    }
	}
	export class UpdateResourceReq {
	    memory: string;
	    cpus: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateResourceReq(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.memory = source["memory"];
	        this.cpus = source["cpus"];
	    }
	}

}


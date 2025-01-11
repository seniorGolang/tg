// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (client-tracer.go at 14.05.2020, 2:13) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

// func (tr *Transport) renderClientTracer(outDir string) (err error) {
//
//	srcFile := newSrc(filepath.Base(outDir))
//	srcFile.PackageComment(doNotEdit)
//
//	srcFile.ImportName(packageHttp, "http")
//	srcFile.ImportName(packageFiber, "fiber")
//	srcFile.ImportName(packageZeroLogLog, "log")
//	srcFile.ImportAlias(packageOpentracing, "otg")
//
//	srcFile.Line().Add(tr.extractSpanClientFunc())
//	srcFile.Line().Add(tr.injectSpanClientFunc())
//
//	return srcFile.Save(path.Join(outDir, "tracer.go"))
// }

// func (tr *Transport) extractSpanClientFunc() Code {
//
//	return Func().Id("extractSpan").Params(Id(_ctx_).Qual(packageContext, "Context"), Id("opName").String()).Params(Id("span").Qual(packageOpentracing, "Span")).Block(
//
//		Line().Var().Id("opts").Op("[]").Qual(packageOpentracing, "StartSpanOption"),
//		Id("span").Op("=").Qual(packageOpentracing, "SpanFromContext").Call(Id(_ctx_)),
//
//		Line().If(Id("span").Op("==").Nil()).Block(
//			Qual(packageZeroLogLog, "Ctx").Call(Id("ctx")).Dot("Debug").Call().Dot("Msg").Call(Lit("context does not contain span")),
//		).Else().Block(
//			Id("opts").Op("=").Append(Id("opts"), Qual(packageOpentracing, "ChildOf").Call(Id("span").Dot("Context").Call())),
//		),
//
//		Line().Id("span").Op("=").Qual(packageOpentracing, "GlobalTracer").Call().Dot("StartSpan").Call(Id("opName"), Id("opts").Op("...")),
//		Return(),
//	)
// }

// func (tr *Transport) injectSpanClientFunc() Code {
//	return Func().Id("injectSpan").Params(Id("span").Qual(packageOpentracing, "Span"), Id("request").Op("*").Qual(packageFiber, "Request")).Params().Block(
//		Id("headers").Op(":=").Make(Qual(packageHttp, "Header")),
//		If(Err().Op(":=").Qual(packageOpentracing, "GlobalTracer").Call().
//			Dot("Inject").Call(
//			Id("span").Dot("Context").Call(),
//			Qual(packageOpentracing, "HTTPHeaders"),
//			Qual(packageOpentracing, "HTTPHeadersCarrier").Call(Id("headers")),
//		).Op(";").Err().Op("!=").Nil()).Block(
//			Id("log").Dot("Warn").Call().Dot("Err").Call(Err()).Dot("Msg").Call(Lit("inject span to HTTP headers")),
//		),
//		For(List(Id("key"), Id("values")).Op(":=").Range().Id("headers")).Block(
//			Id("request").Dot("Header").Dot("Set").Call(Id("key"), Qual(packageStrings, "Join").Call(Id("values"), Lit(";"))),
//		),
//	)
// }

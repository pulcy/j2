// Copyright (c) 2016 Pulcy.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package flags

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestOptionsParse(t *testing.T) {
	Convey("Parse options", t, func() {
		Convey("Simple K=V", func() {
			o := Options{}
			o.Set("key=value")
			v, ok := o.Get("key")
			So(v, ShouldEqual, "value")
			So(ok, ShouldEqual, true)
		})
		Convey("Simple K=\"V\"", func() {
			o := Options{}
			o.Set("key=\"value\"")
			v, ok := o.Get("key")
			So(v, ShouldEqual, "value")
			So(ok, ShouldEqual, true)
		})
		Convey("Simple K=Int Array", func() {
			o := Options{}
			o.Set("key=[5, 6]")
			v, ok := o.Get("key")
			So(v, ShouldResemble, []interface{}{5, 6})
			So(ok, ShouldEqual, true)
		})
		Convey("Simple K=String Array", func() {
			o := Options{}
			o.Set("key=[\"Hello\", \"World\"]")
			v, ok := o.Get("key")
			So(v, ShouldResemble, []interface{}{"Hello", "World"})
			So(ok, ShouldEqual, true)
		})
		Convey("Simple K=Map", func() {
			o := Options{}
			o.Set("key={ pet=\"kitten\" foo=\"monkey\" }")
			v, ok := o.Get("key")
			So(v, ShouldResemble, map[string]interface{}{"pet": "kitten", "foo": "monkey"})
			So(ok, ShouldEqual, true)
		})
	})
}

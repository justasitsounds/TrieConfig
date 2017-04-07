package trieconfig

import (
	"bytes"
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sections config", func() {

	Context("Unmarshalling a single Section", func() {

		DescribeTable("should use the section.Name as a resource identifier",
			func(js, resourceId string) {
				var section Section
				Expect(json.Unmarshal([]byte(js), &section)).ToNot(HaveOccurred())
				Expect(section.ResourceID).To(Equal(resourceId))
			},
			Entry("no resourceId in json", `{"name":"root"}`, "root"),
			Entry("no resourceId, name includes a space", `{"name":"news 24"}`, "news_24"),
			Entry("no resourceId, name includes UpperSpace chars", `{"name":"NEWS 24"}`, "news_24"),
			Entry("resourceId, does not get overriden by name", `{"name":"news", "resourceid":"olds"}`, "olds"),
			Entry("resourceId, includes space", `{"name":"news", "resourceid":"ol ds"}`, "ol_ds"),
			Entry("resourceId, includes UpperSpace chars", `{"name":"news", "resourceid":"OLdS"}`, "olds"),
		)

		It("Should have SectionType:Collection by default", func() {
			var js = `{
			"name":"section"
		}`
			var section Section
			Expect(json.Unmarshal([]byte(js), &section)).ToNot(HaveOccurred())
			Expect(section.SectionType).To(Equal(COLLECTION))
		})

		It("should allow for mapping to arbitrary types", func() {
			var js = `{
			"name":"section",
			"imgurl":"http://somewhere",
			"foo":{
				"bar":"baz"
			}
		}`

			var section Section
			Expect(json.Unmarshal([]byte(js), &section)).ToNot(HaveOccurred())

			var arbitrary struct {
				Name   string
				Imgurl string
				Foo    struct {
					Bar string
				}
			}

			Expect(section.Map(&arbitrary)).To(Not(HaveOccurred()))
			Expect(arbitrary.Imgurl).To(Equal("http://somewhere"))
			Expect(arbitrary.Foo.Bar).To(Equal("baz"))
		})
	})

	Describe("Nested sections", func() {
		var js = `{
		"name":"root",
		"type":"item",
		"section":[
		{
			"name":"section1"
		},
		{
			"name":"section2"
		}
		]
	}`
		var (
			section Section
			err     error
		)

		BeforeEach(func() {
			err = json.Unmarshal([]byte(js), &section)
		})
		Context("Unmarshalling", func() {

			It("Should unmarshal without error", func() {
				Expect(err).ToNot(HaveOccurred())
			})

			It("root section should have two children", func() {
				Expect(len(section.Children)).To(Equal(2))
				Expect(section.SectionType).To(Equal(ITEM))
			})

			It("Should unmarshal the section Types properly", func() {
				Expect(section.Children[0].SectionType).To(Equal(COLLECTION))
			})
		})

		Context("Traversing nested Sections", func() {
			It("Should fetch by resourceRoute", func() {
				kid1 := section.Traverse([]string{"section1"})
				Expect(kid1).To(BeAssignableToTypeOf(&Section{}))
				Expect(kid1.Name).To(Equal("section1"))
			})
		})
	})
})

var _ = Describe("ConfigGetter", func() {
	Context("Constructor", func() {
		var js = `{ "name":"root", "type":"item", "section":[ { "name":"applications" } ] }`
		var config bytes.Buffer
		config.WriteString(js)

		It("should not throw an error", func() {
			_, err := NewConfigGetter(&config)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CurrentStatus string

var (
	StatusFailure CurrentStatus = "Failure"
	StatusSuccess CurrentStatus = "Success"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// InvoiceSpec defines the desired state of Invoice
type InvoiceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	InvoiceData InvoiceData `json:"invoiceData" yaml:"invoiceData"`
}

// InvoiceStatus defines the observed state of Invoice
type InvoiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	LastProcessedTime  *metav1.Time `json:"lastProcessedTime,omitempty"`
	ObservedGeneration int64        `json:"observedGeneration,omitempty"`
	// Current phase of the operator.
	Status   CurrentStatus `json:"currentStatus,omitempty"`
	Endpoint string        `json:"endpoint,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Invoice is the Schema for the invoices API
type Invoice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InvoiceSpec   `json:"spec,omitempty"`
	Status InvoiceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// InvoiceList contains a list of Invoice
type InvoiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Invoice `json:"items"`
}

type InvoiceData struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Number    string  `json:"number" yaml:"number"`
	IssueDate string  `json:"issueDate" yaml:"issueDate"`
	SaleDate  string  `json:"saleDate" yaml:"saleDate"`
	DueDate   string  `json:"dueDate" yaml:"dueDate"`
	Notes     string  `json:"notes" yaml:"notes"`
	Company   Company `json:"company" yaml:"company"`
	Bank      Bank    `json:"bank" yaml:"bank"`
	Items     []*Item `json:"items" yaml:"items"`
	Currency  string  `json:"currency" yaml:"currency"`
	Signature string  `json:"signature" yaml:"signature"`
	Options   Options `json:"options,omitempty" yaml:"options,omitempty"`
}

// Company details of buyer and seller.
type Company struct {
	Buyer  Buyer  `json:"buyer" yaml:"buyer"`
	Seller Seller `json:"seller" yaml:"seller"`
}

// Buyer company details.
type Buyer struct {
	Name    string `json:"name" yaml:"name"`
	Address string `json:"address" yaml:"address"`
	VAT     string `json:"vat" yaml:"vat"`
}

// Seller company details.
type Seller struct {
	Name    string `json:"name" yaml:"name"`
	Address string `json:"address" yaml:"address"`
	VAT     string `json:"vat" yaml:"vat"`
}

// Bank details on the invoice.
type Bank struct {
	AccountNumber string `json:"accountNumber" yaml:"accountNumber"`
	Swift         string `json:"swift" yaml:"swift"`
}

// Item parameters.
type Item struct {
	Description string  `json:"description" yaml:"description"`
	Quantity    float64 `json:"quantity" yaml:"quantity"`
	UnitPrice   float64 `json:"unitPrice" yaml:"unitPrice"`
	VATRate     float64 `json:"vatRate" yaml:"vatRate"`
}

// Options of the PDF document.
type Options struct {
	FontFamily string `json:"font" yaml:"font" default:"Arial,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Invoice{}, &InvoiceList{})
}
